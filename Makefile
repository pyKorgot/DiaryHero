APP_NAME := diaryhero
MAIN_PACKAGE := ./cmd/diaryhero
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)

.PHONY: help run build test fmt tidy clean reset-db docker-build docker-up docker-down docker-logs

help:
	@printf "Available targets:\n"
	@printf "  make run       - run the service locally\n"
	@printf "  make build     - build the binary into $(BIN_PATH)\n"
	@printf "  make test      - run Go tests\n"
	@printf "  make fmt       - format Go code\n"
	@printf "  make tidy      - tidy Go modules\n"
	@printf "  make clean     - remove build artifacts\n"
	@printf "  make reset-db  - remove local SQLite database\n"
	@printf "  make docker-build - build docker image\n"
	@printf "  make docker-up    - start docker compose locally\n"
	@printf "  make docker-down  - stop docker compose\n"
	@printf "  make docker-logs  - follow docker compose logs\n"

run:
	go run $(MAIN_PACKAGE)

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_PATH) $(MAIN_PACKAGE)

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)

reset-db:
	rm -f data/diaryhero.db data/diaryhero.db-shm data/diaryhero.db-wal

docker-build:
	docker compose build

docker-up:
	mkdir -p data
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f
