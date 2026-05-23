package adapters

import (
	"context"
	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"
)

func GetAdapters(rdb *redis.Client) []Adapter {
	var list []Adapter

	list = append(list, NewLinkedIn())

	if appID, appKey := os.Getenv("ADZUNA_APP_ID"), os.Getenv("ADZUNA_APP_KEY"); appID != "" && appKey != "" {
		list = append(list, NewAdzuna(appID, appKey, "br"))
	} else {
		slog.Warn("ADZUNA_APP_ID ou ADZUNA_APP_KEY não configurados, adapter ignorado")
	}

	list = append(list, NewTheMuse())

	if apiKey := os.Getenv("JOOBLE_API_KEY"); apiKey != "" {
		list = append(list, NewJooble(apiKey, rdb))
	} else {
		slog.Warn("JOOBLE_API_KEY não configurada, adapter ignorado")
	}

	greenhouseSlugs, err := FetchGreenhouseSlugs(context.Background())
	if err != nil {
		slog.Warn("falha ao carregar slugs do Greenhouse", "error", err)
	}
	for _, slug := range greenhouseSlugs {
		list = append(list, NewGreenhouse(slug, slug))
	}

	leverCompanies, err := FetchLeverSlugs(context.Background())
	if err != nil {
		slog.Warn("falha ao carregar empresas do Lever", "error", err)
	}
	for _, c := range leverCompanies {
		list = append(list, NewLever(c.Slug, c.Name))
	}

	return list
}
