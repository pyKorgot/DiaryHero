package sqlite

import (
	"context"
	"database/sql"
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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Hero{}, domain.HeroState{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := seedLocations(ctx, tx); err != nil {
		return domain.Hero{}, domain.HeroState{}, err
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
			l.code,
			l.title
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
		&state.LocationTitle,
	)
	if err != nil {
		return domain.Hero{}, domain.HeroState{}, err
	}

	hero.Gender = "female"
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
		{code: "roadside_rumor", title: "Roadside rumor", weight: 4},
		{code: "cheap_work", title: "Cheap work", weight: 3},
		{code: "tavern_rest", title: "Tavern rest", weight: 3},
		{code: "small_loss", title: "Small loss", weight: 2},
		{code: "small_luck", title: "Small luck", weight: 2},
		{code: "strange_stranger", title: "Strange stranger", weight: 3},
		{code: "missed_meal", title: "Missed meal", weight: 2},
		{code: "rainy_walk", title: "Rainy walk", weight: 3},
		{code: "market_pickup", title: "Market pickup", weight: 3},
		{code: "ferry_crossing", title: "Ferry crossing", weight: 2},
		{code: "campfire_night", title: "Campfire night", weight: 2},
		{code: "temple_errand", title: "Temple errand", weight: 2},
		{code: "forest_detour", title: "Forest detour", weight: 2},
		{code: "dockside_argument", title: "Dockside argument", weight: 2},
		{code: "found_shelter", title: "Found shelter", weight: 2},
		{code: "road_to_new_place", title: "Road to new place", weight: 3},
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

func seedLocations(ctx context.Context, tx *sql.Tx) error {
	locations := []struct {
		code   string
		title  string
		danger int
		tags   string
	}{
		{code: "rivergate", title: "Rivergate", danger: 1, tags: `["town","trade"]`},
		{code: "old_wharf", title: "Old Wharf", danger: 2, tags: `["port","fog"]`},
		{code: "ashgrove", title: "Ashgrove", danger: 2, tags: `["forest","road"]`},
		{code: "sunfield", title: "Sunfield", danger: 1, tags: `["village","fields"]`},
		{code: "wayfarer_rest", title: "Wayfarer's Rest", danger: 1, tags: `["inn","road"]`},
	}

	for _, location := range locations {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO locations (code, title, danger_level, tags_json)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(code) DO NOTHING
		`, location.code, location.title, location.danger, location.tags); err != nil {
			return fmt.Errorf("seed location %s: %w", location.code, err)
		}
	}

	return nil
}
