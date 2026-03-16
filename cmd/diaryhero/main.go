package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"diaryhero/internal/app"
	"diaryhero/internal/config"
	"diaryhero/internal/logging"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := logging.New(cfg.LogLevel)

	application, err := app.New(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to initialize application", "error", err)
		return
	}

	if err := application.Run(ctx); err != nil {
		logger.Error("application stopped with error", "error", err)
	}
}
