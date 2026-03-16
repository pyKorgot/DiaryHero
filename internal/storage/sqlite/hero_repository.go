package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"diaryhero/internal/domain"
)

type HeroRepository struct {
	db *sql.DB
}

func NewHeroRepository(db *sql.DB) *HeroRepository {
	return &HeroRepository{db: db}
}

func (r *HeroRepository) EnsureDefaultHero(ctx context.Context) (domain.Hero, domain.HeroState, error) {
	if hero, state, err := r.GetDefaultHero(ctx); err == nil {
		return hero, state, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.Hero{}, domain.HeroState{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Hero{}, domain.HeroState{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO locations (code, title, danger_level, tags_json)
		VALUES ('rivergate', 'Rivergate', 1, '["town","trade"]')
		ON CONFLICT(code) DO NOTHING
	`); err != nil {
		return domain.Hero{}, domain.HeroState{}, fmt.Errorf("seed location: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO heroes (id, name, archetype, voice_style)
		VALUES (1, 'Mira Vale', 'runaway scribe', 'wry, observant, a little dramatic')
		ON CONFLICT(id) DO NOTHING
	`); err != nil {
		return domain.Hero{}, domain.HeroState{}, fmt.Errorf("seed hero: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO hero_state (hero_id, location_id, health, energy, stress, gold, current_time)
		SELECT 1, locations.id, 100, 70, 15, 12, 'morning'
		FROM locations
		WHERE locations.code = 'rivergate'
		ON CONFLICT(hero_id) DO NOTHING
	`); err != nil {
		return domain.Hero{}, domain.HeroState{}, fmt.Errorf("seed hero state: %w", err)
	}

	if err := seedEventTypes(ctx, tx); err != nil {
		return domain.Hero{}, domain.HeroState{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Hero{}, domain.HeroState{}, fmt.Errorf("commit seed transaction: %w", err)
	}

	return r.GetDefaultHero(ctx)
}

func (r *HeroRepository) GetDefaultHero(ctx context.Context) (domain.Hero, domain.HeroState, error) {
	return r.getHeroAndState(ctx, 1)
}

func (r *HeroRepository) GetHeroByID(ctx context.Context, heroID int64) (domain.Hero, error) {
	hero, _, err := r.getHeroAndState(ctx, heroID)
	if err != nil {
		return domain.Hero{}, err
	}

	return hero, nil
}

func (r *HeroRepository) GetStateByHeroID(ctx context.Context, heroID int64) (domain.HeroState, error) {
	_, state, err := r.getHeroAndState(ctx, heroID)
	if err != nil {
		return domain.HeroState{}, err
	}

	return state, nil
}

func (r *HeroRepository) getHeroAndState(ctx context.Context, heroID int64) (domain.Hero, domain.HeroState, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			h.id,
			h.name,
			h.archetype,
			h.voice_style,
			h.created_at,
			hs.hero_id,
			hs.location_id,
			hs.health,
			hs.energy,
			hs.stress,
			hs.gold,
			hs.current_time,
			hs.updated_at,
			l.code
		FROM heroes h
		JOIN hero_state hs ON hs.hero_id = h.id
		JOIN locations l ON l.id = hs.location_id
		WHERE h.id = ?
	`, heroID)

	var hero domain.Hero
	var state domain.HeroState
	var heroCreatedAt string
	var stateUpdatedAt string

	err := row.Scan(
		&hero.ID,
		&hero.Name,
		&hero.Archetype,
		&hero.VoiceStyle,
		&heroCreatedAt,
		&state.HeroID,
		&state.LocationID,
		&state.Health,
		&state.Energy,
		&state.Stress,
		&state.Gold,
		&state.CurrentTime,
		&stateUpdatedAt,
		&state.LocationCode,
	)
	if err != nil {
		return domain.Hero{}, domain.HeroState{}, err
	}

	hero.CreatedAt = parseSQLiteTime(heroCreatedAt)
	state.UpdatedAt = parseSQLiteTime(stateUpdatedAt)

	return hero, state, nil
}

func parseSQLiteTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed
		}
	}

	return time.Time{}
}

func seedEventTypes(ctx context.Context, tx *sql.Tx) error {
	events := []struct {
		code   string
		title  string
		weight int
	}{
		{code: "roadside_rumor", title: "Roadside rumor", weight: 5},
		{code: "cheap_work", title: "Cheap work", weight: 4},
		{code: "tavern_rest", title: "Tavern rest", weight: 3},
		{code: "small_loss", title: "Small loss", weight: 2},
		{code: "small_luck", title: "Small luck", weight: 2},
		{code: "strange_stranger", title: "Strange stranger", weight: 3},
		{code: "missed_meal", title: "Missed meal", weight: 2},
		{code: "rainy_walk", title: "Rainy walk", weight: 3},
		{code: "borrowed_coin", title: "Borrowed coin", weight: 2},
		{code: "market_pickup", title: "Market pickup", weight: 4},
	}

	for _, event := range events {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO event_types (code, title, base_weight)
			VALUES (?, ?, ?)
			ON CONFLICT(code) DO NOTHING
		`, event.code, event.title, event.weight); err != nil {
			return fmt.Errorf("seed event type %s: %w", event.code, err)
		}
	}

	return nil
}
