APP_NAME    := backend-cp
BIN_DIR     := bin
MAIN_PKG    := ./cmd/server
SWAG        := $(shell go env GOPATH)/bin/swag
MIGRATE_URL ?= postgres://postgres:postgres@localhost:5432/cinema?sslmode=disable

.PHONY: help run build test lint fmt tidy swag migrate-up migrate-down migrate-create docker-up docker-down clean

help: ## Liet ke cac lenh co san
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

run: ## Chay server o che do dev
	go run $(MAIN_PKG)

build: ## Build binary vao bin/
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME) $(MAIN_PKG)

test: ## Chay unit test
	go test ./... -race -count=1

lint: ## Chay go vet (cai golangci-lint de kiem tra sau hon)
	go vet ./...

fmt: ## Format toan bo source
	go fmt ./...

tidy: ## Don dep go.mod / go.sum
	go mod tidy

swag: ## Sinh lai swagger docs vao thu muc docs/
	@command -v $(SWAG) >/dev/null 2>&1 || go install github.com/swaggo/swag/cmd/swag@latest
	$(SWAG) init -g cmd/server/main.go -o docs --parseDependency --parseInternal

migrate-up: ## Chay migration len ban moi nhat
	migrate -path migrations -database "$(MIGRATE_URL)" up

migrate-down: ## Rollback 1 buoc migration
	migrate -path migrations -database "$(MIGRATE_URL)" down 1

migrate-create: ## Tao migration moi: make migrate-create name=add_showtimes
	migrate create -ext sql -dir migrations -seq $(name)

docker-up: ## Chay postgres + backend bang docker compose
	docker compose up -d --build

docker-down: ## Dung va xoa container
	docker compose down

clean: ## Xoa binary da build
	rm -rf $(BIN_DIR)
