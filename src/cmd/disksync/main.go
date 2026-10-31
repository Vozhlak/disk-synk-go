package main

import (
	"context"
	"disk-sync/src/internal/config"
	"disk-sync/src/internal/differ"
	"disk-sync/src/internal/gdrive"
	"disk-sync/src/internal/lib/logger"
	"disk-sync/src/internal/localfs"
	"disk-sync/src/internal/storage"
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

	localFsClient := localfs.New(cfg.LocalDir)

	storages := map[string]storage.Storage{
		"Google Drive":     driveClient,
		"Yandex Disk":      ydiskClient,
		"Local Filesystem": localFsClient,
	}

	listAllFilesToSync(ctx, log, storages)
}

func listAllFilesToSync(ctx context.Context, log *slog.Logger, storages map[string]storage.Storage) {
	localFiles, err := storages["Local Filesystem"].ListFiles(ctx)
	if err != nil {
		log.Error(
			"failed to list local files",
			slog.String("storage", "Local Filesystem"),
			slog.Any("error", err),
		)

		os.Exit(1)
	}

	log.Info(
		"found local files",
		slog.Int("count", len(localFiles)),
	)

	for name, client := range storages {
		if name == "Local Filesystem" {
			continue
		}

		log.Info(
			"calculating diff...",
			slog.String("storage", name),
		)

		cloudFiles, err := client.ListFiles(ctx)
		if err != nil {
			log.Error(
				"failed to list files",
				slog.String("storage", name),
				slog.Any("error", err),
			)

			os.Exit(1)
		}

		filesToSync := differ.Diff(localFiles, cloudFiles)

		log.Info(
			"diff calculated",
			slog.String("storage", name),
			slog.Int("files_to_sync", len(filesToSync)),
		)

		for _, file := range filesToSync {
			log.Info(
				" - File to sync",
				slog.String("storage", name),
				slog.String("name", file.Name),
			)
		}
	}
}

func listAllFiles(ctx context.Context, log *slog.Logger, storages map[string]storage.Storage) {
	for name, client := range storages {
		files, err := client.ListFiles(ctx)
		if err != nil {
			log.Error(
				"failed to list files",
				slog.String("storage", name),
				slog.Any("error", err),
			)

			os.Exit(1)
		}

		log.Info("found files",
			slog.String("storage", name),
			slog.Int("count", len(files)),
		)

		for _, file := range files {
			log.Info(
				" - File",
				slog.String("name", file.Name),
				slog.String("type", file.Type),
			)
		}
	}
}

func checkYandexDiskAccess(ctx context.Context, client *ydisk.Client) error {
	return client.CheckAccess(ctx)
}

func checkDriveAccess(ctx context.Context, client *gdrive.Client) error {
	return client.CheckAccess(ctx)
}
