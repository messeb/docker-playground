.DEFAULT_GOAL := help

.PHONY: demo clean help

demo: ## Start all services and run the consumer — watch the circuit breaker live
	docker compose up -d traefik api
	docker compose up --build consumer
	docker compose down

clean: ## Stop containers and remove images
	docker compose down --rmi local

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-8s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
