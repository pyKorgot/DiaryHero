package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"diaryhero/internal/config"
	"diaryhero/internal/narrator"
	"diaryhero/internal/openrouter"
	"diaryhero/internal/sim"
	"diaryhero/internal/storage/sqlite"
	"diaryhero/internal/telegram"
	"diaryhero/internal/worker"
)

type App struct {
	config    config.Config
	logger    *slog.Logger
	db        *sql.DB
	orClient  *openrouter.Client
	tgBot     *telegram.Bot
	scheduler *worker.Scheduler
}

func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	db, err := sqlite.Open(ctx, cfg.DatabasePath)
	if err != nil {
		return nil, err
	}

	heroRepo := sqlite.NewHeroRepository(db)
	if _, _, err := heroRepo.EnsureDefaultHero(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ensure default hero: %w", err)
	}

	tickRepo := sqlite.NewTickRepository(db)
	journalRepo := sqlite.NewJournalRepository(db)
	simRepo := sqlite.NewSimRepository(db)
	orClient := openrouter.NewClient(cfg.OpenRouter)
	tgBot, err := telegram.New(cfg.Telegram, logger)
	if err != nil {
		db.Close()
		return nil, err
	}
	simulator := sim.NewEngine(heroRepo, simRepo)
	narrationService := narrator.New(orClient)
	processor := worker.NewTickProcessor(logger, tickRepo, simulator, narrationService, journalRepo, tgBot)
	scheduler := worker.NewScheduler(logger, heroRepo, tickRepo, processor, cfg.TickInterval)

	return &App{
		config:    cfg,
		logger:    logger,
		db:        db,
		orClient:  orClient,
		tgBot:     tgBot,
		scheduler: scheduler,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("starting diaryhero", "env", a.config.AppEnv, "database_path", a.config.DatabasePath)
	if a.orClient.Enabled() {
		a.logger.Info(
			"openrouter client configured",
			"base_url", a.config.OpenRouter.BaseURL,
			"primary_model", a.config.OpenRouter.PrimaryModel,
			"fallback_model", a.config.OpenRouter.FallbackModel,
			"timeout", a.config.OpenRouter.Timeout.String(),
		)
	} else {
		a.logger.Warn("openrouter api key is not configured; narration remains disabled")
	}

	if a.tgBot != nil && a.tgBot.Enabled() {
		a.logger.Info("telegram bot configured", "mode", a.config.Telegram.Mode, "default_chat_id", a.config.Telegram.DefaultChatID)
		if err := a.tgBot.Start(ctx); err != nil {
			return err
		}
	} else {
		a.logger.Warn("telegram bot token is not configured; telegram delivery remains disabled")
	}

	if err := a.scheduler.Start(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	a.logger.Info("shutting down diaryhero")

	if err := a.db.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}

	return nil
}
