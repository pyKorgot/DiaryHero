package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"diaryhero/internal/domain"
)

type TickProcessor struct {
	logger    *slog.Logger
	tickRepo  domain.TickRepository
	simulator domain.Simulator
	narrator  domain.Narrator
	journal   domain.JournalRepository
	publisher domain.Publisher
}

func NewTickProcessor(logger *slog.Logger, tickRepo domain.TickRepository, simulator domain.Simulator, narrator domain.Narrator, journal domain.JournalRepository, publisher domain.Publisher) *TickProcessor {
	return &TickProcessor{
		logger:    logger,
		tickRepo:  tickRepo,
		simulator: simulator,
		narrator:  narrator,
		journal:   journal,
		publisher: publisher,
	}
}

func (p *TickProcessor) Process(ctx context.Context, tick domain.Tick) error {
	startedAt := time.Now().UTC()
	if err := p.tickRepo.MarkStarted(ctx, tick.ID, startedAt); err != nil {
		return fmt.Errorf("mark tick started: %w", err)
	}

	result, err := p.simulator.RunTick(ctx, tick)
	if err != nil {
		finishedAt := time.Now().UTC()
		markErr := p.tickRepo.MarkFailed(ctx, tick.ID, finishedAt, err.Error())
		if markErr != nil {
			p.logger.Error("failed to mark tick as failed", "tick_id", tick.ID, "error", markErr)
		}
		return err
	}

	narrative, err := p.narrator.GenerateEntry(ctx, domain.NarrativeInput{
		Hero:       result.Hero,
		HeroState:  result.HeroState,
		EventType:  result.EventType,
		WorldEvent: result.WorldEvent,
	})
	if err != nil {
		finishedAt := time.Now().UTC()
		markErr := p.tickRepo.MarkFailed(ctx, tick.ID, finishedAt, err.Error())
		if markErr != nil {
			p.logger.Error("failed to mark tick as failed", "tick_id", tick.ID, "error", markErr)
		}
		return fmt.Errorf("generate entry: %w", err)
	}

	entry, err := p.journal.CreateGenerated(ctx, tick.HeroID, result.WorldEvent.ID, narrative.Text)
	if err != nil {
		finishedAt := time.Now().UTC()
		markErr := p.tickRepo.MarkFailed(ctx, tick.ID, finishedAt, err.Error())
		if markErr != nil {
			p.logger.Error("failed to mark tick as failed", "tick_id", tick.ID, "error", markErr)
		}
		return fmt.Errorf("store journal entry: %w", err)
	}

	finishedAt := time.Now().UTC()
	if err := p.tickRepo.MarkCompleted(ctx, tick.ID, finishedAt); err != nil {
		return fmt.Errorf("mark tick completed: %w", err)
	}

	p.logger.Info(
		"tick processed",
		"tick_id", tick.ID,
		"hero_id", tick.HeroID,
		"event_code", result.WorldEvent.EventCode,
		"event_title", result.EventType.Title,
		"gold", result.HeroState.Gold,
		"energy", result.HeroState.Energy,
		"stress", result.HeroState.Stress,
		"health", result.HeroState.Health,
		"current_time", result.HeroState.CurrentTime,
		"journal_entry_id", entry.ID,
		"narration_source", narrative.Source,
		"narration_model", narrative.Model,
	)

	p.logger.Info(
		"event details",
		"tick_id", tick.ID,
		"summary", formatEventSummary(result),
		"payload", result.WorldEvent.PayloadJSON,
		"outcome", result.WorldEvent.OutcomeJSON,
	)

	p.logger.Info(
		"journal entry generated",
		"tick_id", tick.ID,
		"journal_entry_id", entry.ID,
		"text", narrative.Text,
	)

	if p.publisher != nil && p.publisher.Enabled() {
		if err := p.publisher.PublishText(ctx, narrative.Text); err != nil {
			p.logger.Warn("failed to publish journal entry to telegram", "tick_id", tick.ID, "error", err)
		} else {
			p.logger.Info("journal entry published to telegram", "tick_id", tick.ID, "journal_entry_id", entry.ID)
		}
	}

	return nil
}

func formatEventSummary(result domain.TickResult) string {
	parts := []string{
		fmt.Sprintf("event=%s", result.EventType.Title),
		fmt.Sprintf("time=%s", result.HeroState.CurrentTime),
		fmt.Sprintf("health=%d", result.HeroState.Health),
		fmt.Sprintf("energy=%d", result.HeroState.Energy),
		fmt.Sprintf("stress=%d", result.HeroState.Stress),
		fmt.Sprintf("gold=%d", result.HeroState.Gold),
	}

	return strings.Join(parts, " | ")
}
