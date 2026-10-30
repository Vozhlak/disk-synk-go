package main

import (
	"context"
	"disk-sync/src/internal/config"
	"disk-sync/src/internal/gdrive"
	"disk-sync/src/internal/lib/logger"
	"disk-sync/src/internal/ydisk"
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

	ydiskClient, err := ydisk.New(
		ctx,
		cfg.YandexDisk.ClientID,
		cfg.YandexDisk.ClientSecret,
		cfg.YandexDisk.TokenPath,
	)
	if err != nil {
		log.Error(
			"failed to authenticate with Yandex Disk",
			slog.String("token_path", cfg.YandexDisk.TokenPath),
			slog.String("error", err.Error()),
		)

		os.Exit(1)
	}

	log.Info("Yandex Disk client initialized")

	if err = checkYandexDiskAccess(ctx, ydiskClient); err != nil {
		log.Error(
			"Yandex Disk access check failed",
			slog.String("error", err.Error()),
		)

		os.Exit(1)
	}

	log.Info("successfully authenticated with Yandex Disk")

	ydiskFiles, err := ydiskClient.ListFiles(ctx)
	if err != nil {
		log.Error(
			"failed to list yandex disk files",
			slog.Any("error", err),
		)

		return
	}

	log.Info(
		"found Yandex Disk files",
		slog.Int("count", len(ydiskFiles)),
	)

	for _, file := range ydiskFiles {
		log.Info(
			" - File",
			slog.String("name", file.Name),
		)
	}
}

func checkYandexDiskAccess(ctx context.Context, client *ydisk.Client) error {
	return client.CheckAccess(ctx)
}

func checkDriveAccess(ctx context.Context, client *gdrive.Client) error {
	return client.CheckAccess(ctx)
}
