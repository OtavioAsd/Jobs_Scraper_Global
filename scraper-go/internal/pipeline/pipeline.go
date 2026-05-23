package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Benevanio/Jobs_Scraper_Global/scraper-go/internal/adapters"
	"github.com/Benevanio/Jobs_Scraper_Global/scraper-go/internal/dedup"
	"github.com/Benevanio/Jobs_Scraper_Global/scraper-go/internal/jobstore"
	"github.com/Benevanio/Jobs_Scraper_Global/scraper-go/internal/models"
	"github.com/redis/go-redis/v9"
)

const defaultMaxConcurrency = 150

type result struct {
	jobs []models.Job
	err  error
}

func Run(ctx context.Context, adapterList []adapters.Adapter, req models.ScrapeRequest) []models.Job {
	maxConcurrency := req.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = defaultMaxConcurrency
	}

	type task struct {
		adapter adapters.Adapter
		keyword string
	}

	tasks := make([]task, 0, len(adapterList)*len(req.Keywords))
	for _, a := range adapterList {
		for _, kw := range req.Keywords {
			tasks = append(tasks, task{adapter: a, keyword: kw})
		}
	}

	sem := make(chan struct{}, maxConcurrency)
	results := make(chan result, len(tasks))
	var wg sync.WaitGroup

	for _, t := range tasks {
		wg.Add(1)
		sem <- struct{}{}

		go func(t task) {
			defer wg.Done()
			defer func() { <-sem }()

			jobs, err := t.adapter.Search(ctx, t.keyword, req)
			results <- result{jobs: jobs, err: err}

			if err != nil {
				slog.Warn("adapter falhou",
					"source", t.adapter.SourceName(),
					"keyword", t.keyword,
					"error", err,
				)
				return
			}

			slog.Info("adapter concluído",
				"source", t.adapter.SourceName(),
				"keyword", t.keyword,
				"count", len(jobs),
			)
		}(t)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allJobs []models.Job
	for r := range results {
		if r.err == nil {
			allJobs = append(allJobs, r.jobs...)
		}
	}

	return dedup.DedupeJobs(allJobs)
}

func IndexJobsInValkey(ctx context.Context, rdb *redis.Client, jobs []models.Job, keywords []string) {
	if rdb == nil || len(jobs) == 0 {
		return
	}

	globalIndexKey := "scraper:jobs:index"
	indexTTL := 72 * time.Hour

	// Limpa os índices de keyword anteriores (composto + sub-termos)
	for _, kw := range keywords {
		sanitizedKw := strings.ToLower(strings.TrimSpace(kw))
		if sanitizedKw == "" {
			continue
		}

		normalizedForDel := strings.ReplaceAll(sanitizedKw, "/", " ")
		normalizedForDel = strings.Join(strings.Fields(normalizedForDel), " ")
		rdb.Del(ctx, fmt.Sprintf("scraper:jobs:keyword:%s", normalizedForDel))

		for _, term := range strings.Fields(normalizedForDel) {
			rdb.Del(ctx, fmt.Sprintf("scraper:jobs:keyword:%s", term))
		}
	}

	for _, job := range jobs {
		// ✅ Usa o ID estável como membro de todos os Sets,
		// igual ao que jobstore.SaveBatch faz — nunca a URL bruta
		id := jobstore.StableID(&job)
		if id == "" {
			continue
		}

		// Índice global: apenas IDs (sem duplicar com URLs)
		rdb.SAdd(ctx, globalIndexKey, id)

		titleLower := strings.ToLower(job.Title)
		descLower := strings.ToLower(job.Description)

		for _, kw := range keywords {
			sanitizedKw := strings.ToLower(strings.TrimSpace(kw))
			if sanitizedKw == "" {
				continue
			}

			normalizedKw := strings.ReplaceAll(sanitizedKw, "/", " ")
			subTerms := strings.Fields(normalizedKw)

			// Keyword composta: indexa só se todos os sub-termos aparecem na vaga
			matchAll := true
			for _, term := range subTerms {
				if !strings.Contains(titleLower, term) && !strings.Contains(descLower, term) {
					matchAll = false
					break
				}
			}

			if matchAll {
				fullKey := fmt.Sprintf("scraper:jobs:keyword:%s", strings.Join(subTerms, " "))
				rdb.SAdd(ctx, fullKey, id)
				rdb.Expire(ctx, fullKey, indexTTL)
			}

			// Sub-termos individuais: indexa cada um independentemente
			for _, term := range subTerms {
				if term == "" {
					continue
				}
				if strings.Contains(titleLower, term) || strings.Contains(descLower, term) {
					termKey := fmt.Sprintf("scraper:jobs:keyword:%s", term)
					rdb.SAdd(ctx, termKey, id)
					rdb.Expire(ctx, termKey, indexTTL)
				}
			}
		}
	}

	rdb.Expire(ctx, globalIndexKey, indexTTL)
	slog.Info("Valkey invertido atualizado", "total_vagas", len(jobs))
}
