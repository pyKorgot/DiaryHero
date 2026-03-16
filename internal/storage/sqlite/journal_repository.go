package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"diaryhero/internal/domain"
)

type JournalRepository struct {
	db *sql.DB
}

func NewJournalRepository(db *sql.DB) *JournalRepository {
	return &JournalRepository{db: db}
}

func (r *JournalRepository) CreateGenerated(ctx context.Context, heroID int64, worldEventID int64, text string) (domain.JournalEntry, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO journal_entries (hero_id, world_event_id, channel, text, status)
		VALUES (?, ?, 'internal', ?, 'generated')
	`, heroID, worldEventID, text)
	if err != nil {
		return domain.JournalEntry{}, fmt.Errorf("insert journal entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return domain.JournalEntry{}, fmt.Errorf("read journal entry id: %w", err)
	}

	return domain.JournalEntry{
		ID:           id,
		HeroID:       heroID,
		WorldEventID: worldEventID,
		Channel:      "internal",
		Text:         text,
		Status:       "generated",
		CreatedAt:    time.Now().UTC(),
	}, nil
}
