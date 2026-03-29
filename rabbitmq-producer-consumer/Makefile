.DEFAULT_GOAL := help

.PHONY: demo logs open-ui clean help

demo: ## Build images and stream producer + consumer logs in parallel (CTRL+C to stop)
	docker compose up --build -d rabbitmq
	docker compose up --build consumer producer

logs: ## Follow live logs for all services (CTRL+C to stop)
	docker compose logs -f

open-ui: ## Open the RabbitMQ management UI in the browser
	open http://localhost:15672

clean: ## Stop all containers and remove built images
	docker compose down --rmi local

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-10s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
