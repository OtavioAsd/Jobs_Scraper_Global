package pipeline

import (
	"context"
	"log/slog"

	"github.com/Benevanio/Jobs_Scraper_Global/scraper-go/internal/adapters"
	"github.com/Benevanio/Jobs_Scraper_Global/scraper-go/internal/models"
	"github.com/redis/go-redis/v9"
)

type SearchConfig struct {
	Keywords       []string `json:"keywords"`
	SearchLocation string   `json:"searchLocation"`
	JobTypes       string   `json:"jobTypes"`
	TimeFilter     string   `json:"timeFilter"`
	RemoteOnly     bool     `json:"remoteOnly"`
	MaxConcurrency int      `json:"maxConcurrency"`
}

func ScrapeAllSources(
	ctx context.Context,
	config SearchConfig,
	rdb *redis.Client,
) ([]models.Job, error) {
	slog.Info("starting scrape", "keywords", config.Keywords)

	adapterList := adapters.GetAdapters(rdb)

	req := models.ScrapeRequest{
		Keywords:       config.Keywords,
		SearchLocation: config.SearchLocation,
		JobTypes:       config.JobTypes,
		TimeFilter:     config.TimeFilter,
		RemoteOnly:     config.RemoteOnly,
		MaxConcurrency: config.MaxConcurrency,
	}

	jobs := Run(ctx, adapterList, req)

	slog.Info("scrape finished",
		"total_jobs", len(jobs),
		"keywords", len(config.Keywords),
		"adapters", len(adapterList),
	)

	return jobs, nil
}
