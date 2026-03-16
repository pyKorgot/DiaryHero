package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"diaryhero/internal/domain"
	"diaryhero/internal/sim"
)

type SimRepository struct {
	db *sql.DB
}

func NewSimRepository(db *sql.DB) *SimRepository {
	return &SimRepository{db: db}
}

var _ sim.Repository = (*SimRepository)(nil)

func (r *SimRepository) ListEventTypes(ctx context.Context) ([]domain.EventType, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, code, title, base_weight, cooldown_ticks
		FROM event_types
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("query event types: %w", err)
	}
	defer rows.Close()

	var eventTypes []domain.EventType
	for rows.Next() {
		var eventType domain.EventType
		if err := rows.Scan(&eventType.ID, &eventType.Code, &eventType.Title, &eventType.BaseWeight, &eventType.CooldownTicks); err != nil {
			return nil, fmt.Errorf("scan event type: %w", err)
		}
		eventTypes = append(eventTypes, eventType)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate event types: %w", err)
	}

	return eventTypes, nil
}

func (r *SimRepository) CreateWorldEventAndApplyState(ctx context.Context, tick domain.Tick, eventType domain.EventType, payload map[string]any, nextState domain.HeroState, outcome map[string]any) (domain.WorldEvent, domain.HeroState, error) {
	payloadJSON, err := sim.MarshalJSON(payload)
	if err != nil {
		return domain.WorldEvent{}, domain.HeroState{}, fmt.Errorf("encode payload: %w", err)
	}

	outcomeJSON, err := sim.MarshalJSON(outcome)
	if err != nil {
		return domain.WorldEvent{}, domain.HeroState{}, fmt.Errorf("encode outcome: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.WorldEvent{}, domain.HeroState{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO world_events (hero_id, tick_id, event_type_id, payload_json, outcome_json)
		VALUES (?, ?, ?, ?, ?)
	`, tick.HeroID, tick.ID, eventType.ID, payloadJSON, outcomeJSON)
	if err != nil {
		return domain.WorldEvent{}, domain.HeroState{}, fmt.Errorf("insert world event: %w", err)
	}

	worldEventID, err := result.LastInsertId()
	if err != nil {
		return domain.WorldEvent{}, domain.HeroState{}, fmt.Errorf("read world event id: %w", err)
	}

	updatedAt := nextState.UpdatedAt.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, `
		UPDATE hero_state
		SET location_id = ?, health = ?, energy = ?, stress = ?, gold = ?, current_time = ?, updated_at = ?
		WHERE hero_id = ?
	`, nextState.LocationID, nextState.Health, nextState.Energy, nextState.Stress, nextState.Gold, nextState.CurrentTime, updatedAt, tick.HeroID); err != nil {
		return domain.WorldEvent{}, domain.HeroState{}, fmt.Errorf("update hero state: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.WorldEvent{}, domain.HeroState{}, fmt.Errorf("commit tick transaction: %w", err)
	}

	return domain.WorldEvent{
		ID:          worldEventID,
		HeroID:      tick.HeroID,
		TickID:      tick.ID,
		EventTypeID: eventType.ID,
		EventCode:   eventType.Code,
		PayloadJSON: payloadJSON,
		OutcomeJSON: outcomeJSON,
		CreatedAt:   time.Now().UTC(),
	}, nextState, nil
}
