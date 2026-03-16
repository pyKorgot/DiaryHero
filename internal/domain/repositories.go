package domain

import (
	"context"
	"time"
)

type HeroRepository interface {
	EnsureDefaultHero(ctx context.Context) (Hero, HeroState, error)
	GetDefaultHero(ctx context.Context) (Hero, HeroState, error)
	GetHeroByID(ctx context.Context, heroID int64) (Hero, error)
	GetStateByHeroID(ctx context.Context, heroID int64) (HeroState, error)
}

type TickRepository interface {
	CreateScheduled(ctx context.Context, heroID int64, scheduledFor time.Time) (Tick, error)
	MarkStarted(ctx context.Context, tickID int64, startedAt time.Time) error
	MarkCompleted(ctx context.Context, tickID int64, finishedAt time.Time) error
	MarkFailed(ctx context.Context, tickID int64, finishedAt time.Time, errorText string) error
}

type Simulator interface {
	RunTick(ctx context.Context, tick Tick) (TickResult, error)
}

type Narrator interface {
	GenerateEntry(ctx context.Context, input NarrativeInput) (NarrativeOutput, error)
}

type JournalRepository interface {
	CreateGenerated(ctx context.Context, heroID int64, worldEventID int64, text string) (JournalEntry, error)
}

type Publisher interface {
	Enabled() bool
	PublishText(ctx context.Context, text string) error
}
