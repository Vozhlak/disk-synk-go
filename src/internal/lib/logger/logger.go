package logger

import (
	"log/slog"
	"os"
)

func New(env string) *slog.Logger {
	switch env {
	case "local":
		return slog.New(slog.NewTextHandler(os.Stdout, nil))
	case "dev", "prod":
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	default:
		return slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
}
