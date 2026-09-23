.PHONY: dev down logs ps build prod

dev: ## Full stack local (build + up)
	docker compose up -d --build

build: ## Build images only
	docker compose build

prod: ## Prod overlay preview (needs IMAGE_TAG, e.g. make prod IMAGE_TAG=abc123)
	IMAGE_TAG=$${IMAGE_TAG:?set IMAGE_TAG} docker compose -f docker-compose.yml -f docker-compose.prod.yml config --quiet

down: ## Stop stack (keep volumes)
	docker compose down

logs: ## Follow all logs
	docker compose logs -f

ps: ## Container status
	docker compose ps
