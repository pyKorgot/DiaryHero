# AGENTS.md

## Project Overview

`DiaryHero` is a Go service that simulates a single fantasy hero, creates world ticks on a schedule, generates a first-person diary entry, stores it in SQLite, and optionally publishes it to Telegram.

Current flow:

```text
Scheduler -> Tick -> Simulation -> WorldEvent -> Narrator -> JournalEntry -> Telegram
```

## Current Tech Stack

- Go
- SQLite via `modernc.org/sqlite`
- Scheduler via `github.com/robfig/cron/v3`
- LLM generation via direct HTTP client to `OpenRouter`
- Telegram integration via `github.com/go-telegram/bot`

## Important Paths

- `cmd/diaryhero` - entrypoint
- `internal/app` - application wiring and lifecycle
- `internal/config` - env loading and config structs
- `internal/domain` - domain models and interfaces
- `internal/sim` - tick simulation engine
- `internal/narrator` - diary text generation
- `internal/openrouter` - OpenRouter client
- `internal/telegram` - Telegram bot and publisher
- `internal/storage/sqlite` - DB open, migrations, repositories
- `internal/worker` - scheduler and tick processor

## How To Run

- `make run` - run locally
- `make test` - run tests
- `make fmt` - format Go code
- `make build` - build binary
- `make reset-db` - remove local SQLite db

Config is loaded from `.env` automatically by `internal/config/config.go`.

## Environment Variables

Core:

- `APP_ENV`
- `LOG_LEVEL`
- `DATABASE_PATH`
- `TICK_INTERVAL`

OpenRouter:

- `OPENROUTER_BASE_URL`
- `OPENROUTER_API_KEY`
- `OPENROUTER_PRIMARY_MODEL`
- `OPENROUTER_FALLBACK_MODEL`
- `OPENROUTER_SITE_URL`
- `OPENROUTER_APP_NAME`
- `OPENROUTER_TIMEOUT`

Telegram:

- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_DEFAULT_CHAT_ID`
- `TELEGRAM_MODE`

## Behavior Notes

- If `OPENROUTER_API_KEY` is missing, narration falls back to a local stub template.
- If `TELEGRAM_BOT_TOKEN` is missing, Telegram bot startup is skipped.
- If `TELEGRAM_DEFAULT_CHAT_ID` is empty, the app still runs, but publishing logs a warning instead of failing the tick.
- Default local tick interval is intentionally short for development (`15s` unless overridden).

## Database Notes

The SQLite database is created locally under `data/` by default.

Current important tables:

- `heroes`
- `locations`
- `hero_state`
- `event_types`
- `ticks`
- `world_events`
- `journal_entries`
- `outbox` (reserved for later delivery pipeline work)

Migrations live in `internal/storage/sqlite/migrations`.

## Implementation Conventions

- Keep changes small and incremental.
- Prefer repository-backed domain flow over ad-hoc SQL in workers.
- Keep `worker` package orchestration-focused; business logic should live in `sim`, `narrator`, `telegram`, or repositories.
- Keep narration resilient: if external generation fails, degrade gracefully when possible.
- Do not introduce heavy infrastructure unless it clearly serves the MVP.
- Preserve current MVP scope: one hero, one scheduler, one local database.

## When Editing

- Run `make fmt` and `make test` after code changes.
- If behavior changes materially, update `README.md`.
- If config changes, update both `.env.example` and `README.md`.
- If schema changes, add or update migrations instead of patching the database manually.

## Telegram Notes

- `/start` currently replies with the detected `chat_id`.
- `/chatid` returns the current chat id.
- For channels, the bot must be added as admin with permission to post.
- For groups, the bot needs permission to send messages.

## Good Next Steps

- Add outbox-based Telegram delivery and retries
- Persist discovered Telegram chat ids in SQLite
- Add memory/continuity layer
- Add anti-repeat event logic
- Add tests for simulation and narration fallback behavior
