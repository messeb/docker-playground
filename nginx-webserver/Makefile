.DEFAULT_GOAL := help

.PHONY: up down restart logs ps open headers help

up: ## Start the container in the background
	docker compose up -d
	@echo "\n  Server    http://localhost\n"

down: ## Stop and remove the container
	docker compose down

restart: ## Restart the container
	docker compose restart

logs: ## Follow nginx access and error logs
	docker compose logs -f

ps: ## Show container status
	docker compose ps

open: ## Open the webserver in the browser
	open http://localhost

headers: ## Show response headers for key endpoints
	@echo "── / ──────────────────────────────────────────"
	@curl -sI http://localhost | grep -E "HTTP|Content-Type|Cache-Control|X-|Referrer|Content-Encoding"
	@echo "── /static/style.css ──────────────────────────"
	@curl -sI http://localhost/static/style.css | grep -E "HTTP|Content-Type|Cache-Control|Expires|Content-Encoding"
	@echo "── /health ────────────────────────────────────"
	@curl -s http://localhost/health && echo ""

help: ## Show this help message
	@echo "Usage: make <target>\n"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-12s %s\n", $$1, $$2 }' Makefile
	@echo ""
