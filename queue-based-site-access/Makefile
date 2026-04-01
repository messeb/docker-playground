.DEFAULT_GOAL := help

.PHONY: up demo down logs ps clean \
        admin-capacity admin-release admin-status \
        help

# ── Lifecycle ──────────────────────────────────────────────────────────────────

up: ## Build images and start all services
	docker compose up --build -d
	@echo ""
	@echo "  Protected site  →  http://localhost"
	@echo ""
	@echo "  Admin API:"
	@echo "    make admin-capacity N=3   # set max concurrent users"
	@echo "    make admin-release        # release one active slot"
	@echo "    make admin-status         # show live queue state"
	@echo ""

demo: up ## Start services and print demo walkthrough
	@echo "  Demo walkthrough:"
	@echo "  1. Open http://localhost in multiple browser tabs / incognito windows"
	@echo "  2. First $(N) sessions land on the target page; the rest land on the queue"
	@echo "  3. The queue page shows your position and (after 5 admissions) an ETA"
	@echo "  4. Run 'make admin-release' to free a slot — the next user auto-redirects"
	@echo "  5. Run 'make admin-capacity N=5' to allow more concurrent users"
	@echo ""

down: ## Stop and remove all containers
	docker compose down

logs: ## Follow logs for all services (CTRL+C to stop)
	docker compose logs -f

ps: ## Show container status
	docker compose ps

clean: ## Stop containers and remove locally built images
	docker compose down --rmi local

# ── Admin ──────────────────────────────────────────────────────────────────────

N ?= 3

admin-capacity: ## Set max concurrent users — usage: make admin-capacity N=5
	@curl -sf -X POST http://localhost/api/admin/capacity \
	  -H "Content-Type: application/json" \
	  -d '{"capacity": $(N)}' | python3 -m json.tool

admin-release: ## Release one active slot and admit the next queued user
	@curl -sf -X POST http://localhost/api/admin/release \
	  -H "Content-Type: application/json" \
	  -d '{"count": 1}' | python3 -m json.tool

admin-status: ## Show live queue state (capacity, active, queued)
	@curl -sf http://localhost/api/admin/status | python3 -m json.tool

# ── Help ───────────────────────────────────────────────────────────────────────

help: ## Show available make targets
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-22s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Variables:"
	@echo "  N=<number>             Used by admin-capacity (default: 3)"
	@echo ""
