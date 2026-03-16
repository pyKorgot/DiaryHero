package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"diaryhero/internal/domain"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron         *cron.Cron
	logger       *slog.Logger
	tickRepo     domain.TickRepository
	heroRepo     domain.HeroRepository
	processor    *TickProcessor
	tickInterval time.Duration
}

func NewScheduler(logger *slog.Logger, heroRepo domain.HeroRepository, tickRepo domain.TickRepository, processor *TickProcessor, tickInterval time.Duration) *Scheduler {
	return &Scheduler{
		cron:         cron.New(),
		logger:       logger,
		tickRepo:     tickRepo,
		heroRepo:     heroRepo,
		processor:    processor,
		tickInterval: tickInterval,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	spec := fmt.Sprintf("@every %s", s.tickInterval)
	if _, err := s.cron.AddFunc(spec, func() {
		if err := s.scheduleTick(context.Background()); err != nil {
			s.logger.Error("failed to schedule tick", "error", err)
		}
	}); err != nil {
		return fmt.Errorf("register cron job: %w", err)
	}

	s.cron.Start()
	s.logger.Info("scheduler started", "interval", s.tickInterval.String())

	go func() {
		<-ctx.Done()
		stopCtx := s.cron.Stop()
		select {
		case <-stopCtx.Done():
		case <-time.After(5 * time.Second):
			s.logger.Warn("scheduler stop timed out")
		}
	}()

	return s.scheduleTick(ctx)
}

func (s *Scheduler) scheduleTick(ctx context.Context) error {
	hero, _, err := s.heroRepo.GetDefaultHero(ctx)
	if err != nil {
		return fmt.Errorf("load default hero: %w", err)
	}

	tick, err := s.tickRepo.CreateScheduled(ctx, hero.ID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("create tick: %w", err)
	}

	s.logger.Info("tick scheduled", "tick_id", tick.ID, "hero_id", tick.HeroID, "scheduled_for", tick.ScheduledFor.Format(time.RFC3339))

	if err := s.processor.Process(ctx, tick); err != nil {
		return fmt.Errorf("process tick %d: %w", tick.ID, err)
	}

	return nil
}
