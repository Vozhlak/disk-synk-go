package main

import (
	"disk-sync/src/internal/config"
	"disk-sync/src/internal/lib/logger"
	"log/slog"
	"os"
)

const configPath = "config.yaml"

func main() {
	cfg, err := config.Load(configPath)
	if err != nil {
		log := logger.New("local")

		log.Error(
			"failed to load configuration",
			slog.String("path", configPath),
			slog.String("error", err.Error()),
		)

		os.Exit(1)
		return
	}

	log := logger.New(cfg.Env)

	log.Info(
		"configuration loaded successfully",
		slog.String("path", configPath),
		slog.String("env", cfg.Env),
	)
}
