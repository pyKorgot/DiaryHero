package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"diaryhero/internal/domain"
)

type TickRepository struct {
	db *sql.DB
}

func NewTickRepository(db *sql.DB) *TickRepository {
	return &TickRepository{db: db}
}

func (r *TickRepository) CreateScheduled(ctx context.Context, heroID int64, scheduledFor time.Time) (domain.Tick, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO ticks (hero_id, scheduled_for, status)
		VALUES (?, ?, 'scheduled')
	`, heroID, scheduledFor.UTC().Format(time.RFC3339))
	if err != nil {
		return domain.Tick{}, fmt.Errorf("insert tick: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return domain.Tick{}, fmt.Errorf("read tick id: %w", err)
	}

	return domain.Tick{
		ID:           id,
		HeroID:       heroID,
		ScheduledFor: scheduledFor.UTC(),
		Status:       "scheduled",
		CreatedAt:    time.Now().UTC(),
	}, nil
}

func (r *TickRepository) MarkStarted(ctx context.Context, tickID int64, startedAt time.Time) error {
	if _, err := r.db.ExecContext(ctx, `
		UPDATE ticks
		SET started_at = ?, status = 'running', error_text = NULL
		WHERE id = ?
	`, startedAt.UTC().Format(time.RFC3339), tickID); err != nil {
		return fmt.Errorf("update tick started state: %w", err)
	}

	return nil
}

func (r *TickRepository) MarkCompleted(ctx context.Context, tickID int64, finishedAt time.Time) error {
	if _, err := r.db.ExecContext(ctx, `
		UPDATE ticks
		SET finished_at = ?, status = 'completed', error_text = NULL
		WHERE id = ?
	`, finishedAt.UTC().Format(time.RFC3339), tickID); err != nil {
		return fmt.Errorf("update tick completed state: %w", err)
	}

	return nil
}

func (r *TickRepository) MarkFailed(ctx context.Context, tickID int64, finishedAt time.Time, errorText string) error {
	if _, err := r.db.ExecContext(ctx, `
		UPDATE ticks
		SET finished_at = ?, status = 'failed', error_text = ?
		WHERE id = ?
	`, finishedAt.UTC().Format(time.RFC3339), errorText, tickID); err != nil {
		return fmt.Errorf("update tick failed state: %w", err)
	}

	return nil
}
