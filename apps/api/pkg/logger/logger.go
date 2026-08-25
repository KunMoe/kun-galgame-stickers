package logger

import (
	"log/slog"
	"os"
)

func Init(mode string) {
	level := slog.LevelInfo
	if mode == "dev" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
}
