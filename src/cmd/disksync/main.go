package main

import (
	"context"
	"disk-sync/src/internal/config"
	"disk-sync/src/internal/gdrive"
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
	}

	log := logger.New(cfg.Env)

	log.Info(
		"configuration loaded successfully",
		slog.String("path", configPath),
		slog.String("env", cfg.Env),
	)

	ctx := context.Background()

	driveClient, err := gdrive.New(
		ctx,
		cfg.GoogleDrive.ClientID,
		cfg.GoogleDrive.ClientSecret,
		cfg.GoogleDrive.TokenPath,
	)
	if err != nil {
		log.Error(
			"failed to authenticate with Google Drive",
			slog.String("token_path", cfg.GoogleDrive.TokenPath),
			slog.String("error", err.Error()),
		)

		os.Exit(1)
	}

	log.Info("Google Drive client initialized")

	if err = checkDriveAccess(ctx, driveClient); err != nil {
		log.Error(
			"Google Drive access check failed",
			slog.String("error", err.Error()),
		)

		os.Exit(1)
	}

	log.Info("Google Drive access check completed successfully")

	files, err := driveClient.ListFiles(ctx)
	if err != nil {
		log.Error(
			"failed to list files",
			slog.Any("error", err),
		)

		os.Exit(1)
	}

	log.Info(
		"found Google Drive files",
		slog.Int("count", len(files)),
	)

	for _, file := range files {
		log.Info(
			" - File",
			slog.String("name", file.Name),
		)
	}
}

func checkDriveAccess(ctx context.Context, client *gdrive.Client) error {
	return client.CheckAccess(ctx)
}
