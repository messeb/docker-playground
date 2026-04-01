.PHONY: help compose shutdown restart clean logs ps

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

compose: ## Start all services (builds images if needed)
	docker compose up -d --build

shutdown: ## Stop all services
	docker compose down

restart: ## Restart all services
	docker compose restart

clean: ## Stop all services and remove volumes
	docker compose down -v

logs: ## Follow logs for all services
	docker compose logs -f

ps: ## Show running containers and their status
	docker compose ps
