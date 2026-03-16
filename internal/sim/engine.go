package sim

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"diaryhero/internal/domain"
)

type Repository interface {
	ListEventTypes(ctx context.Context) ([]domain.EventType, error)
	CreateWorldEventAndApplyState(ctx context.Context, tick domain.Tick, eventType domain.EventType, payload map[string]any, nextState domain.HeroState, outcome map[string]any) (domain.WorldEvent, domain.HeroState, error)
}

type Engine struct {
	heroRepo domain.HeroRepository
	repo     Repository
	rng      *rand.Rand
}

func NewEngine(heroRepo domain.HeroRepository, repo Repository) *Engine {
	return &Engine{
		heroRepo: heroRepo,
		repo:     repo,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *Engine) RunTick(ctx context.Context, tick domain.Tick) (domain.TickResult, error) {
	hero, err := e.heroRepo.GetHeroByID(ctx, tick.HeroID)
	if err != nil {
		return domain.TickResult{}, fmt.Errorf("load hero: %w", err)
	}

	currentState, err := e.heroRepo.GetStateByHeroID(ctx, tick.HeroID)
	if err != nil {
		return domain.TickResult{}, fmt.Errorf("load hero state: %w", err)
	}

	eventTypes, err := e.repo.ListEventTypes(ctx)
	if err != nil {
		return domain.TickResult{}, fmt.Errorf("list event types: %w", err)
	}

	if len(eventTypes) == 0 {
		return domain.TickResult{}, fmt.Errorf("no event types configured")
	}

	selected := chooseWeightedEvent(e.rng, eventTypes)
	nextState, outcome := applyEvent(currentState, selected)
	payload := map[string]any{
		"location_code": currentState.LocationCode,
		"current_time":  currentState.CurrentTime,
		"event_code":    selected.Code,
	}

	worldEvent, persistedState, err := e.repo.CreateWorldEventAndApplyState(ctx, tick, selected, payload, nextState, outcome)
	if err != nil {
		return domain.TickResult{}, fmt.Errorf("persist tick results: %w", err)
	}

	return domain.TickResult{
		Hero:       hero,
		Tick:       tick,
		EventType:  selected,
		WorldEvent: worldEvent,
		HeroState:  persistedState,
	}, nil
}

func chooseWeightedEvent(rng *rand.Rand, eventTypes []domain.EventType) domain.EventType {
	totalWeight := 0
	for _, eventType := range eventTypes {
		weight := eventType.BaseWeight
		if weight < 1 {
			weight = 1
		}
		totalWeight += weight
	}

	roll := rng.Intn(totalWeight)
	running := 0
	for _, eventType := range eventTypes {
		weight := eventType.BaseWeight
		if weight < 1 {
			weight = 1
		}
		running += weight
		if roll < running {
			return eventType
		}
	}

	return eventTypes[len(eventTypes)-1]
}

func applyEvent(state domain.HeroState, eventType domain.EventType) (domain.HeroState, map[string]any) {
	next := state
	next.UpdatedAt = time.Now().UTC()
	next.CurrentTime = advanceTime(state.CurrentTime)

	goldDelta := 0
	energyDelta := -2
	stressDelta := 0
	healthDelta := 0

	switch eventType.Code {
	case "cheap_work":
		goldDelta = 4
		energyDelta = -8
		stressDelta = 2
	case "tavern_rest":
		goldDelta = -2
		energyDelta = 10
		stressDelta = -4
	case "small_loss":
		goldDelta = -3
		stressDelta = 4
	case "small_luck":
		goldDelta = 5
		stressDelta = -2
	case "strange_stranger":
		stressDelta = 3
	case "missed_meal":
		energyDelta = -10
		healthDelta = -2
		stressDelta = 2
	case "rainy_walk":
		energyDelta = -5
		stressDelta = -1
	case "borrowed_coin":
		goldDelta = 2
		stressDelta = 1
	case "market_pickup":
		goldDelta = -1
		energyDelta = -1
		stressDelta = -1
	default:
		stressDelta = 1
	}

	next.Gold = clampMin(state.Gold+goldDelta, 0)
	next.Energy = clamp(state.Energy+energyDelta, 0, 100)
	next.Stress = clamp(state.Stress+stressDelta, 0, 100)
	next.Health = clamp(state.Health+healthDelta, 0, 100)

	return next, map[string]any{
		"event_code":   eventType.Code,
		"gold_delta":   goldDelta,
		"energy_delta": energyDelta,
		"stress_delta": stressDelta,
		"health_delta": healthDelta,
		"next_time":    next.CurrentTime,
	}
}

func advanceTime(current string) string {
	cycle := []string{"morning", "afternoon", "evening", "night"}
	for index, value := range cycle {
		if value == current {
			return cycle[(index+1)%len(cycle)]
		}
	}

	return cycle[0]
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func clampMin(value, minValue int) int {
	if value < minValue {
		return minValue
	}
	return value
}

func MarshalJSON(value map[string]any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}
