package sim

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"diaryhero/internal/domain"
)

type Repository interface {
	ListEventTypes(ctx context.Context) ([]domain.EventType, error)
	CreateWorldEventAndApplyState(ctx context.Context, tick domain.Tick, eventType domain.EventType, payload map[string]any, nextState domain.HeroState, outcome map[string]any) (domain.WorldEvent, domain.HeroState, error)
}

type eventDefinition struct {
	sceneTemplates []string
	goldDelta      int
	energyDelta    int
	stressDelta    int
	healthDelta    int
	moveTargets    []string
	details        []string
}

type locationDefinition struct {
	title string
	mood  string
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
	nextState, payload, outcome := e.applyEvent(currentState, selected)

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

func (e *Engine) applyEvent(state domain.HeroState, eventType domain.EventType) (domain.HeroState, map[string]any, map[string]any) {
	definition, ok := eventDefinitions[eventType.Code]
	if !ok {
		definition = eventDefinition{
			sceneTemplates: []string{"Случилась очередная мелочь, которую еще придется переварить."},
			energyDelta:    -2,
			stressDelta:    1,
		}
	}

	next := state
	next.UpdatedAt = time.Now().UTC()
	next.CurrentTime = advanceTime(state.CurrentTime)

	fromLocationCode := state.LocationCode
	fromLocationTitle := state.LocationTitle
	if fromLocationTitle == "" {
		fromLocationTitle = humanizeCode(fromLocationCode)
	}

	if len(definition.moveTargets) > 0 && e.rng.Intn(100) < 55 {
		targetCode := definition.moveTargets[e.rng.Intn(len(definition.moveTargets))]
		if targetLocation, ok := locations[targetCode]; ok {
			next.LocationCode = targetCode
			next.LocationTitle = targetLocation.title
			next.LocationID = lookupLocationID(targetCode, state.LocationID)
		}
	}

	next.Gold = clampMin(state.Gold+definition.goldDelta, 0)
	next.Energy = clamp(state.Energy+definition.energyDelta, 0, 100)
	next.Stress = clamp(state.Stress+definition.stressDelta, 0, 100)
	next.Health = clamp(state.Health+definition.healthDelta, 0, 100)

	locationMood := ""
	if location, ok := locations[next.LocationCode]; ok {
		locationMood = location.mood
	}

	scene := pick(definition.sceneTemplates, e.rng)
	detail := pick(definition.details, e.rng)

	payload := map[string]any{
		"event_code":          eventType.Code,
		"from_location_code":  fromLocationCode,
		"from_location_title": fromLocationTitle,
		"to_location_code":    next.LocationCode,
		"to_location_title":   next.LocationTitle,
		"time_of_day":         next.CurrentTime,
		"scene":               scene,
		"detail":              detail,
		"location_mood":       locationMood,
	}

	outcome := map[string]any{
		"event_code":      eventType.Code,
		"scene":           scene,
		"detail":          detail,
		"moved_location":  fromLocationCode != next.LocationCode,
		"location_mood":   locationMood,
		"state_direction": describeStateShift(definition),
		"next_time":       next.CurrentTime,
	}

	return next, payload, outcome
}

func describeStateShift(definition eventDefinition) string {
	parts := make([]string, 0, 3)
	if definition.energyDelta <= -6 {
		parts = append(parts, "tired")
	} else if definition.energyDelta >= 6 {
		parts = append(parts, "rested")
	}
	if definition.stressDelta >= 3 {
		parts = append(parts, "uneasy")
	} else if definition.stressDelta <= -3 {
		parts = append(parts, "lighter")
	}
	if definition.healthDelta < 0 {
		parts = append(parts, "worse for wear")
	}
	if len(parts) == 0 {
		return "mixed"
	}
	return strings.Join(parts, ", ")
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

func pick(values []string, rng *rand.Rand) string {
	if len(values) == 0 {
		return ""
	}
	return values[rng.Intn(len(values))]
}

func humanizeCode(code string) string {
	parts := strings.Split(code, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, " ")
}

func lookupLocationID(code string, fallback int64) int64 {
	switch code {
	case "rivergate":
		return 1
	case "old_wharf":
		return 2
	case "ashgrove":
		return 3
	case "sunfield":
		return 4
	case "wayfarer_rest":
		return 5
	default:
		return fallback
	}
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

var locations = map[string]locationDefinition{
	"rivergate":     {title: "Rivergate", mood: "river town with damp stone embankments"},
	"old_wharf":     {title: "Old Wharf", mood: "creaking piers, tar, gulls, and fog"},
	"ashgrove":      {title: "Ashgrove", mood: "a roadside grove full of damp leaves and ash bark"},
	"sunfield":      {title: "Sunfield", mood: "quiet cottages, fences, and wind over the fields"},
	"wayfarer_rest": {title: "Wayfarer's Rest", mood: "an inn yard with wagons, horses, and warm kitchen light"},
}

var eventDefinitions = map[string]eventDefinition{
	"roadside_rumor": {
		sceneTemplates: []string{"A rumor caught me on the road before I could slip away from it.", "Someone leaned in with a rumor that sounded too detailed to be harmless."},
		energyDelta:    -2,
		stressDelta:    1,
		details:        []string{"It mentioned a missing ferryman.", "The name of an old debt surfaced again.", "A caravan was said to have vanished near the grove."},
	},
	"cheap_work": {
		sceneTemplates: []string{"I took ugly work because refusing it would have been even uglier.", "The sort of task nobody boasts about still ended up in my hands."},
		goldDelta:      3,
		energyDelta:    -8,
		stressDelta:    2,
		details:        []string{"Mostly lifting crates and swallowing my pride.", "The pay was mean, but the foreman at least stopped shouting by the end.", "It left my shoulders humming and my temper thin."},
	},
	"tavern_rest": {
		sceneTemplates: []string{"For once I found a corner warm enough to let my guard down.", "A tavern bench and a bowl of something hot almost passed for mercy."},
		goldDelta:      -1,
		energyDelta:    8,
		stressDelta:    -4,
		moveTargets:    []string{"wayfarer_rest", "rivergate"},
		details:        []string{"The room smelled of cloves and wet wool.", "Nobody asked my name twice, which helped.", "A fiddler played badly enough to become comforting."},
	},
	"small_loss": {
		sceneTemplates: []string{"Something small went missing, which is often worse than losing something grand.", "I turned my pockets inside out and still came up short."},
		stressDelta:    4,
		details:        []string{"A ribbon, a note, maybe only my patience.", "I cannot prove who took it, which is the most irritating part.", "The loss was petty enough to feel insulting."},
	},
	"small_luck": {
		sceneTemplates: []string{"Luck brushed my sleeve and moved on before anyone noticed.", "The day briefly remembered I exist and decided not to bite."},
		energyDelta:    2,
		stressDelta:    -2,
		details:        []string{"A door opened at the right moment.", "Someone pointed me toward the dry path instead of the muddy one.", "A stranger answered kindness with kindness, which still surprises me."},
	},
	"strange_stranger": {
		sceneTemplates: []string{"A stranger looked at me as if we shared a secret I had forgotten.", "I crossed paths with someone whose silence said more than speech."},
		stressDelta:    3,
		details:        []string{"They wore travel dust and a ring with the crest filed off.", "They knew my name too quickly.", "They asked one harmless question and left me suspicious anyway."},
	},
	"missed_meal": {
		sceneTemplates: []string{"The day ran long and my stomach noticed before I did.", "By the time I stopped moving, food had become a memory rather than a plan."},
		energyDelta:    -9,
		healthDelta:    -1,
		stressDelta:    2,
		details:        []string{"The smell of bread from somewhere else did not help.", "I told myself I was too busy to eat and almost believed it.", "Hunger makes every noise sound personal."},
	},
	"rainy_walk": {
		sceneTemplates: []string{"Rain found me before shelter did.", "The road turned slick and gray and would not let me pass unnoticed."},
		energyDelta:    -4,
		stressDelta:    -1,
		moveTargets:    []string{"old_wharf", "ashgrove", "sunfield"},
		details:        []string{"My hem carried half the puddles with it.", "Everything smelled of wet bark and river mud.", "At least the rain kept chatter to a minimum."},
	},
	"market_pickup": {
		sceneTemplates: []string{"The market handed me a small useful thing at the exact right moment.", "Among the stalls and elbows I found something worth carrying onward."},
		energyDelta:    -1,
		stressDelta:    -1,
		details:        []string{"A wrapped loaf, a tin cup, and a little goodwill.", "A trader slipped me advice with the purchase.", "The whole place smelled of apples, rope, and impatience."},
	},
	"borrowed_coin": {
		sceneTemplates: []string{"I accepted a small kindness and disliked needing it more than the kindness itself.", "A borrowed coin landed in my palm with more weight than metal should carry."},
		stressDelta:    1,
		details:        []string{"The lender smiled as if that made the debt lighter.", "I promised to repay it sooner than I probably should have.", "Even help can feel sharp when pride gets in the way."},
	},
	"ferry_crossing": {
		sceneTemplates: []string{"I crossed the water with my thoughts in worse order than the current.", "The ferry creaked under us as if it had opinions about every passenger aboard."},
		energyDelta:    -2,
		stressDelta:    1,
		moveTargets:    []string{"old_wharf", "rivergate", "sunfield"},
		details:        []string{"The boatman watched the banks more than the river.", "A child laughed at the spray while the adults stared into their own troubles.", "For a moment the world seemed held together by rope and habit."},
	},
	"campfire_night": {
		sceneTemplates: []string{"Night went softer around a campfire and a few borrowed voices.", "I spent the dark by a fire that asked nothing from me except staying a while."},
		energyDelta:    4,
		stressDelta:    -3,
		moveTargets:    []string{"ashgrove", "wayfarer_rest"},
		details:        []string{"Sparks went up like tiny lies no one bothered to challenge.", "Someone passed around tea too bitter to complain about.", "The sort of warmth that reminds a person how tired she has been."},
	},
	"temple_errand": {
		sceneTemplates: []string{"I ended up carrying a quiet errand for people too polite to call it a favor.", "The temple had one small task and somehow it became my afternoon."},
		energyDelta:    -3,
		stressDelta:    -1,
		moveTargets:    []string{"sunfield", "rivergate"},
		details:        []string{"Incense clung to my sleeves afterward.", "It was the sort of silence that made even clumsy footsteps feel rude.", "The old woman at the door thanked me like she knew more than she said."},
	},
	"forest_detour": {
		sceneTemplates: []string{"The road bent and I let the forest decide the rest for a while.", "I took the longer path through the trees because the ordinary road looked too obvious."},
		energyDelta:    -5,
		stressDelta:    1,
		moveTargets:    []string{"ashgrove", "sunfield"},
		details:        []string{"Every branch seemed to whisper after I passed.", "The ground was soft with old leaves and older doubts.", "I heard something move parallel to me and never saw it."},
	},
	"dockside_argument": {
		sceneTemplates: []string{"A quarrel by the docks spilled wider than it meant to and splashed onto me as well.", "Raised voices at the waterfront have a way of collecting bystanders like driftwood."},
		energyDelta:    -3,
		stressDelta:    3,
		moveTargets:    []string{"old_wharf", "rivergate"},
		details:        []string{"Mostly bluster, but one man had his hand too close to a knife.", "I kept my face calm and my feet ready.", "It ended without blood, which counts as grace in places like that."},
	},
	"found_shelter": {
		sceneTemplates: []string{"By dusk I found shelter before the weather or my luck turned worse.", "A roof, even a doubtful one, looked like a blessing tonight."},
		energyDelta:    6,
		stressDelta:    -2,
		moveTargets:    []string{"wayfarer_rest", "sunfield", "rivergate"},
		details:        []string{"The shutters rattled but held.", "It was not home, only enough wall between me and the dark.", "The bedding smelled of hay and old smoke, and I accepted both."},
	},
	"road_to_new_place": {
		sceneTemplates: []string{"I did not stay put; the road tugged harder than comfort today.", "By the time the light shifted, I had already left one place behind for another."},
		energyDelta:    -4,
		stressDelta:    1,
		moveTargets:    []string{"old_wharf", "ashgrove", "sunfield", "wayfarer_rest", "rivergate"},
		details:        []string{"It felt better to keep moving than to explain myself to walls.", "Travel leaves less room for brooding, though not none.", "The road was kind enough to remain only a road for once."},
	},
}
