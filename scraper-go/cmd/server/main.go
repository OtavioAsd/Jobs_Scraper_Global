package main

import (
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	loadEnv()

	// newRedisClient() está em server.go — usa ParseURL corretamente
	// e valida a conexão com Ping antes de retornar.
	rdb, err := newRedisClient()
	if err != nil {
		slog.Warn("valkey indisponível, Jooble vai operar sem controle de cota", "error", err)
		rdb = nil
	}
	if rdb != nil {
		defer rdb.Close()
	}

	adapterList := buildAdapters(rdb)
	slog.Info("servidor inicializado", "adapters_total", len(adapterList))

	run(adapterList)
}
