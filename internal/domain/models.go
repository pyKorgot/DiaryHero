package domain

import "time"

type Hero struct {
	ID         int64
	Name       string
	Archetype  string
	Gender     string
	VoiceStyle string
	CreatedAt  time.Time
}

type NarrativeInput struct {
	Hero       Hero
	HeroState  HeroState
	EventType  EventType
	WorldEvent WorldEvent
}

type NarrativeOutput struct {
	Text   string
	Source string
	Model  string
}

type HeroState struct {
	HeroID        int64
	LocationID    int64
	Health        int
	Energy        int
	Stress        int
	Gold          int
	CurrentTime   string
	UpdatedAt     time.Time
	LocationCode  string
	LocationTitle string
}

type Tick struct {
	ID           int64
	HeroID       int64
	ScheduledFor time.Time
	Status       string
	CreatedAt    time.Time
}

type EventType struct {
	ID            int64
	Code          string
	Title         string
	BaseWeight    int
	CooldownTicks int
}

type WorldEvent struct {
	ID          int64
	HeroID      int64
	TickID      int64
	EventTypeID int64
	EventCode   string
	PayloadJSON string
	OutcomeJSON string
	CreatedAt   time.Time
}

type TickResult struct {
	Hero       Hero
	Tick       Tick
	EventType  EventType
	WorldEvent WorldEvent
	HeroState  HeroState
}

type JournalEntry struct {
	ID                int64
	HeroID            int64
	WorldEventID      int64
	Channel           string
	ExternalMessageID string
	Text              string
	Status            string
	PublishedAt       *time.Time
	CreatedAt         time.Time
}

type TelegramChat struct {
	ChatID    string
	ChatType  string
	Title     string
	Username  string
	IsDefault bool
	Source    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
