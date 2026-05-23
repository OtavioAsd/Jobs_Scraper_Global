package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Benevanio/Jobs_Scraper_Global/scraper-go/internal/models"
	"github.com/redis/go-redis/v9"
)

const (
	joobleQuotaKey    = "jooble:quota:daily"     // contador no Valkey
	joobleQuotaLimit  = 300                      // chamadas máximas por dia
	joobleSlotSize    = 37                       // keywords por execução
	joobleRotationKey = "jooble:rotation:offset" // offset rotativo no Valkey
)

type JoobleAdapter struct {
	apiKey string
	client *http.Client
	rdb    *redis.Client
}

func NewJooble(apiKey string, rdb *redis.Client) *JoobleAdapter {
	return &JoobleAdapter{
		apiKey: apiKey,
		client: &http.Client{Timeout: 15 * time.Second},
		rdb:    rdb,
	}
}

func (a *JoobleAdapter) SourceName() string { return "Jooble" }

// quotaRemaining retorna quantas chamadas ainda cabem hoje.
// Retorna 0 se o Valkey falhar (fail-safe: não estoura a cota).
func (a *JoobleAdapter) quotaRemaining(ctx context.Context) int {
	used, err := a.rdb.Get(ctx, joobleQuotaKey).Int()
	if err == redis.Nil {
		return joobleQuotaLimit // chave não existe ainda → dia novo
	}
	if err != nil {
		return 0 // Valkey indisponível → para silenciosamente
	}
	remaining := joobleQuotaLimit - used
	if remaining < 0 {
		return 0
	}
	return remaining
}

// incrementQuota registra 1 chamada usada.
// Na primeira chamada do dia, define TTL até meia-noite.
func (a *JoobleAdapter) incrementQuota(ctx context.Context) {
	pipe := a.rdb.Pipeline()
	pipe.Incr(ctx, joobleQuotaKey)

	// TTL dinâmico: segundos restantes até meia-noite
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	ttl := time.Until(midnight)
	pipe.Expire(ctx, joobleQuotaKey, ttl)

	pipe.Exec(ctx) //nolint:errcheck — falha aqui não deve parar o scraper
}

// nextSlot retorna o subconjunto rotacionado de keywords para essa execução.
func (a *JoobleAdapter) nextSlot(ctx context.Context, keywords []string, slotSize int) []string {
	if len(keywords) == 0 {
		return nil
	}

	// Lê offset atual; se não existir começa do 0
	offset, err := a.rdb.Get(ctx, joobleRotationKey).Int()
	if err != nil {
		offset = 0
	}

	// Salva próximo offset para a execução seguinte
	nextOffset := (offset + slotSize) % len(keywords)
	a.rdb.Set(ctx, joobleRotationKey, nextOffset, 0) //nolint:errcheck

	// Monta o slice rotacionado (wrap-around circular)
	result := make([]string, 0, slotSize)
	for i := 0; i < slotSize && i < len(keywords); i++ {
		result = append(result, keywords[(offset+i)%len(keywords)])
	}
	return result
}

// SearchBatch é o ponto de entrada — recebe todas as keywords mas roda
// apenas o slot rotacionado, respeitando a cota diária.
// Retorna os jobs encontrados e para silenciosamente se a cota acabar.
func (a *JoobleAdapter) SearchBatch(ctx context.Context, allKeywords []string, req models.ScrapeRequest) ([]models.Job, error) {
	remaining := a.quotaRemaining(ctx)
	if remaining <= 0 {
		return nil, nil // cota esgotada → para silenciosamente
	}

	slotSize := joobleSlotSize
	if slotSize > remaining {
		slotSize = remaining // não ultrapassa o que sobrou
	}

	slot := a.nextSlot(ctx, allKeywords, slotSize)
	if len(slot) == 0 {
		return nil, nil
	}

	var allJobs []models.Job
	for _, keyword := range slot {
		jobs, err := a.search(ctx, keyword, req)
		if err != nil {
			// 429 ou erro de rede → para silenciosamente, não aborta os outros adapters
			break
		}
		a.incrementQuota(ctx)
		allJobs = append(allJobs, jobs...)
	}

	return allJobs, nil
}

// search faz 1 chamada para 1 keyword (lógica original preservada).
func (a *JoobleAdapter) search(ctx context.Context, keyword string, req models.ScrapeRequest) ([]models.Job, error) {
	endpoint := "https://br.jooble.org/api/" + a.apiKey

	payload := map[string]string{
		"keywords": keyword,
		"location": req.SearchLocation,
	}
	if payload["location"] == "" {
		payload["location"] = "Brasil"
	}

	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("jooble: 429")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jooble: status %d", resp.StatusCode)
	}

	var joobleRes struct {
		Jobs []struct {
			Title    string `json:"title"`
			Location string `json:"location"`
			Snippet  string `json:"snippet"`
			Source   string `json:"source"`
			Type     string `json:"type"`
			Link     string `json:"link"`
			Company  string `json:"company"`
			Updated  string `json:"updated"`
			Salary   string `json:"salary"`
			ID       int64  `json:"id"`
		} `json:"jobs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&joobleRes); err != nil {
		return nil, err
	}

	jobs := make([]models.Job, 0, len(joobleRes.Jobs))
	for _, j := range joobleRes.Jobs {
		jobs = append(jobs, models.Job{
			ID:       j.Link,
			Title:    j.Title,
			Company:  j.Company,
			Location: j.Location,
			URL:      j.Link,
			Salary:   j.Salary,
			Source:   "Jooble",
			Sources:  []string{"Jooble"},
			Keyword:  keyword,
			Keywords: []string{keyword},
		})
	}

	return jobs, nil
}

func (a *JoobleAdapter) Search(ctx context.Context, keyword string, req models.ScrapeRequest) ([]models.Job, error) {
	return a.SearchBatch(ctx, []string{keyword}, req)
}
