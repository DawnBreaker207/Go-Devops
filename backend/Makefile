APP_NAME    := backend-cp
BIN_DIR     := bin
MAIN_PKG    := ./cmd/server
SWAG        := $(shell go env GOPATH)/bin/swag
MIGRATE_URL ?= postgres://postgres:postgres@localhost:5432/cinema?sslmode=disable
DATABASE_NAME ?= cinema
DATABASE_USER ?= postgres

.PHONY: help run build test test-system chaos load test-race lint fmt tidy swag migrate migrate-up migrate-down migrate-seed migrate-db-reset migrate-create migrate-create-seed docker-up docker-down clean

help: ## Liet ke cac lenh co san
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

run: ## Chay server o che do dev
	go run $(MAIN_PKG)

build: ## Build binary vao bin/
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PKG)

test: ## Chay toan bo test (can Postgres + RabbitMQ: docker compose up -d postgres rabbitmq)
	go test ./... -count=1

test-system: ## Bai toan toan thang tren toan bo stack (postgres + rabbitmq) — chay lai referent tests T18/T37/T42/I4
	docker compose up -d postgres rabbitmq
	go test ./... -count=1

chaos: ## Cha race + concurrency + stress (+ T37 seat-sold-once, T18 movie-not-selling) trong Docker (can CGO)
	docker run --rm -v "$(CURDIR):/src" -v "$(shell go env GOMODCACHE):/go/pkg/mod" -w /src \
		-e CGO_ENABLED=1 -e TEST_DB_HOST=host.docker.internal \
		-e QUEUE_TEST_URL=amqp://guest:guest@host.docker.internal:5672/ \
		golang:1.26 go test -race -count=1 -run 'Race|Stress|Concurrent' ./internal/...

load: ## Stress/load nhieu vong: TestStress_NoSeatSoldTwice (T37 single-sale invariant)
	go test -count=1 -run 'TestStress_NoSeatSoldTwice' -timeout 30m ./internal/service/

test-race: ## Chay test voi -race trong Docker (race detector can gcc/CGO)
	docker run --rm -v "$(CURDIR):/src" -v "$(shell go env GOMODCACHE):/go/pkg/mod" -w /src \
		-e CGO_ENABLED=1 -e TEST_DB_HOST=host.docker.internal \
		-e QUEUE_TEST_URL=amqp://guest:guest@host.docker.internal:5672/ \
		golang:1.26 go test -race -count=1 ./...

lint: ## Chay go vet (cai golangci-lint de kiem tra sau hon)
	go vet ./...

fmt: ## Format toan bo source
	go fmt ./...

tidy: ## Don dep go.mod / go.sum
	go mod tidy

swag: ## Sinh lai swagger docs vao thu muc docs/
	@command -v $(SWAG) >/dev/null 2>&1 || go install github.com/swaggo/swag/cmd/swag@latest
	$(SWAG) init -g cmd/server/main.go -o docs --parseDependency --parseInternal

migrate: migrate-up ## Alias apply schema migrations

migrate-up: ## Apply schema migrations
	migrate -path migrations/schema -database "$(MIGRATE_URL)" up

migrate-down: ## Roll back 1 schema step
	migrate -path migrations/schema -database "$(MIGRATE_URL)" down 1

migrate-seed: ## Apply idempotent SQL scripts from migrations/seed (in order)
	@for f in migrations/seed/*.sql; do \
		echo "== applying $$f"; \
		docker compose exec -T postgres psql -U $(DATABASE_USER) -d $(DATABASE_NAME) -v ON_ERROR_STOP=1 < $$f; \
	done

migrate-db-reset: ## Drop & recreate the dev DB, then apply schema + seed (destroys all data!)
	docker compose exec -T postgres psql -U $(DATABASE_USER) -c "DROP DATABASE IF EXISTS $(DATABASE_NAME) WITH (FORCE);" -c "CREATE DATABASE $(DATABASE_NAME);"
	migrate -path migrations/schema -database "$(MIGRATE_URL)" up
	$(MAKE) migrate-seed

migrate-create: ## Scaffold a schema migration: make migrate-create name=f4_halls
	migrate create -ext sql -dir migrations/schema -seq $(name)

migrate-create-seed: ## Scaffold a seed script: make migrate-create-seed name=sample_showtimes
	@n=$$(ls migrations/seed/*.sql 2>/dev/null | wc -l); \
	f=$$(printf '%06d_%s.sql' $$n $$name); \
	touch migrations/seed/$$f; \
	echo created migrations/seed/$$f

docker-up: ## Build and start postgres + rabbitmq + migrate + backend (set secrets or APP_ENV=development in .env)
	docker compose up -d --build

docker-down: ## Dung va xoa container
	docker compose down

clean: ## Xoa binary da build
	rm -rf $(BIN_DIR)
