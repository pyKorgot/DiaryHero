# DiaryHero

`DiaryHero` - это pet project про "живой дневник героя".

Сервис по расписанию делает тик мира, выбирает событие, обновляет состояние героя, генерирует короткую запись от первого лица через `OpenRouter` и может отправлять ее в `Telegram`.

## Что уже работает

- Go-сервис с graceful shutdown
- конфиг через `.env`
- `SQLite` как локальная БД
- миграции и сид одного героя
- scheduler с периодическими тиками
- rule-based simulation engine
- несколько локаций и перемещение героя между ними
- запись `ticks`, `world_events`, `journal_entries`
- генерация текста через `OpenRouter`
- fallback на локальный stub, если LLM недоступна
- Telegram bot c polling для публикации только в каналы
- автоматическое сохранение обнаруженных Telegram-каналов в SQLite
- отправка generated diary entry в Telegram через автоматически обнаруженный channel target

## Текущий поток работы

```text
Scheduler -> Tick -> Simulation -> WorldEvent -> Narrator -> JournalEntry -> Telegram
```

На каждом тике сервис:

1. создает запись в `ticks`
2. выбирает событие из `event_types`
3. при необходимости перемещает героя в новую локацию и обновляет `hero_state`
4. сохраняет `world_events`
5. генерирует текст дневника
6. сохраняет `journal_entries`
7. пытается отправить запись в `Telegram`

## Стек

- `Go`
- `SQLite` через `modernc.org/sqlite`
- `robfig/cron/v3`
- `OpenRouter` через прямой HTTP client
- `go-telegram/bot`

## Структура проекта

```text
cmd/diaryhero          - точка входа
internal/app           - сборка приложения и lifecycle
internal/config        - загрузка env-конфига
internal/domain        - доменные модели и интерфейсы
internal/narrator      - генерация diary entry
internal/openrouter    - клиент OpenRouter
internal/sim           - simulation engine
internal/storage/sqlite - БД, миграции, репозитории
internal/telegram      - Telegram bot и отправка сообщений
internal/worker        - scheduler и tick processing
```

## Быстрый старт

### 1. Подготовить `.env`

Скопируй шаблон и заполни значения:

```bash
cp .env.example .env
```

Минимальный рабочий пример:

```env
APP_ENV=development
LOG_LEVEL=info
DATABASE_PATH=data/diaryhero.db
TICK_INTERVAL=15m

OPENROUTER_BASE_URL=https://openrouter.ai/api/v1
OPENROUTER_API_KEY=
OPENROUTER_PRIMARY_MODEL=openrouter/auto
OPENROUTER_FALLBACK_MODEL=openrouter/auto
OPENROUTER_SITE_URL=
OPENROUTER_APP_NAME=DiaryHero
OPENROUTER_TIMEOUT=30s

TELEGRAM_BOT_TOKEN=
TELEGRAM_MODE=polling
```

Если `OPENROUTER_API_KEY` пустой, сервис будет использовать локальный stub-текст.

Если `TELEGRAM_BOT_TOKEN` не задан, сервис продолжит работать без публикации в Telegram.

### 2. Запустить проект

```bash
make run
```

Сервис сам:

- создаст `SQLite` БД
- применит схему
- засидирует дефолтного героя и набор событий
- начнет создавать тики

## Make-команды

- `make run` - запустить сервис локально
- `make build` - собрать бинарник в `bin/diaryhero`
- `make test` - прогнать тесты
- `make fmt` - форматировать код
- `make tidy` - обновить зависимости
- `make clean` - удалить артефакты сборки
- `make reset-db` - удалить локальную БД

## Docker Compose

Для локального запуска можно использовать `Dockerfile` и `docker-compose.yml` из корня проекта.

### Подготовка

1. Подготовь `.env` рядом с `docker-compose.yml`
2. Заполни как минимум:

```env
APP_ENV=development
LOG_LEVEL=info
TICK_INTERVAL=15m

OPENROUTER_API_KEY=your_openrouter_key
OPENROUTER_PRIMARY_MODEL=openrouter/auto
OPENROUTER_FALLBACK_MODEL=openrouter/auto

TELEGRAM_BOT_TOKEN=your_telegram_bot_token
```

`DATABASE_PATH` вручную задавать не обязательно: в compose он уже направлен в `/app/data/diaryhero.db` внутри контейнера.

Локальная папка `data/` монтируется в контейнер как `/app/data`, поэтому базу удобно смотреть и бэкапить прямо с хоста.

### Запуск

```bash
docker compose up -d --build
```

или через `Makefile`:

```bash
make docker-up
```

### Полезные команды

```bash
docker compose logs -f
docker compose ps
docker compose restart
docker compose down
```

или:

```bash
make docker-logs
make docker-down
```

### Хранение данных

- база `SQLite` хранится в локальной папке `data/`
- при пересборке контейнера данные сохраняются
- для полного сброса можно остановить compose и удалить `data/diaryhero.db*`

## Настройка Telegram

Сейчас бот работает только с каналами.

### Канал

1. Добавь бота в канал как администратора
2. Дай право публиковать сообщения
3. После добавления бот получит membership update и сохранит канал в SQLite
4. Если это первый доступный канал, он станет целью публикации автоматически
5. После этого новые diary entries начнут публиковаться в этот канал автоматически

Если каналов несколько, сервис использует канал, сохраненный в SQLite как default/актуальный target.

## Основные env-переменные

### App

- `APP_ENV`
- `LOG_LEVEL`
- `DATABASE_PATH`
- `TICK_INTERVAL`

### OpenRouter

- `OPENROUTER_BASE_URL`
- `OPENROUTER_API_KEY`
- `OPENROUTER_PRIMARY_MODEL`
- `OPENROUTER_FALLBACK_MODEL`
- `OPENROUTER_SITE_URL`
- `OPENROUTER_APP_NAME`
- `OPENROUTER_TIMEOUT`

### Telegram

- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_MODE`

## Что видно в логах

Во время работы сервис пишет в консоль:

- старт приложения
- конфигурацию `OpenRouter` и `Telegram`
- создание и обработку тиков
- выбранное событие и изменения состояния героя
- сгенерированный текст дневника
- результат попытки отправки в Telegram

## Что дальше можно развивать

- outbox и retry для Telegram delivery
- улучшение выбора target-канала при нескольких подключенных каналах
- memory layer и anti-repeat логика
- outbox и retries для устойчивой отправки в каналы
- admin/debug endpoints
- поддержку нескольких героев
