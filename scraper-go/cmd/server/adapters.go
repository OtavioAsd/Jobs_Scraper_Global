package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/Benevanio/Jobs_Scraper_Global/scraper-go/internal/adapters"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func loadEnv() {
	candidates := []string{
		os.Getenv("ENV_FILE"),
		"/app/.env",
		"../.env",
		".env",
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if err := godotenv.Load(p); err == nil {
			slog.Info("arquivo .env carregado", "path", p)
			return
		}
	}
	slog.Warn("arquivo .env não encontrado, usando variáveis do sistema")
}

func resolveInterfacesPath(filename string) string {
	if v := os.Getenv("INTERFACES_DIR"); v != "" {
		return v + "/" + filename
	}
	candidates := []string{
		"internal/interfaces/" + filename,
		"../internal/interfaces/" + filename,
		"../../internal/interfaces/" + filename,
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "internal/interfaces/" + filename
}

func buildAdapters(rdb *redis.Client) []adapters.Adapter {
	all := make([]adapters.Adapter, 0)

	all = append(all, adapters.NewLinkedIn())

	all = append(all, adapters.NewAdzuna(
		os.Getenv("ADZUNA_APP_ID"),
		os.Getenv("ADZUNA_APP_KEY"),
		"br",
	))

	if err := os.Setenv("GREENHOUSE_COMPANIES_FILE", resolveInterfacesPath("greenhouseCompanies.json")); err != nil {
		slog.Warn("falha ao setar GREENHOUSE_COMPANIES_FILE", "error", err)
	}
	greenhouseSlugs, err := adapters.FetchGreenhouseSlugs(context.Background())
	if err != nil {
		slog.Warn("falha ao buscar slugs do Greenhouse", "error", err)
	}
	for _, slug := range greenhouseSlugs {
		all = append(all, adapters.NewGreenhouse(slug, slug))
	}

	if err := os.Setenv("LEVER_COMPANIES_FILE", resolveInterfacesPath("leverCompanies.json")); err != nil {
		slog.Warn("falha ao setar LEVER_COMPANIES_FILE", "error", err)
	}
	leverCompanies, err := adapters.FetchLeverSlugs(context.Background())
	if err != nil {
		slog.Warn("falha ao carregar leverCompanies.json", "error", err)
	} else {
		for _, c := range leverCompanies {
			all = append(all, adapters.NewLever(c.Slug, c.Name))
		}
	}

	if joobleKey := os.Getenv("JOOBLE_API_KEY"); joobleKey != "" {
		all = append(all, adapters.NewJooble(joobleKey, rdb))
	} else {
		log.Println("AVISO: JOOBLE_API_KEY não encontrada")
	}

	return all
}
