package main

import (
	"log/slog"
	"net"
	"os"
	"time"

	"kun-galgame-sticker-api/internal/app"
	"kun-galgame-sticker-api/pkg/config"
	"kun-galgame-sticker-api/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env")

	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		port := os.Getenv("SERVER_PORT")
		if port == "" {
			port = "9421"
		}
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 2*time.Second)
		if err != nil {
			os.Exit(1)
		}
		_ = conn.Close()
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	application := app.New(cfg)
	addr := ":" + cfg.Server.Port
	slog.Info("listening", "addr", addr)
	if err := application.Fiber.Listen(addr); err != nil {
		slog.Error("listen", "error", err)
		os.Exit(1)
	}
}
