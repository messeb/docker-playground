.DEFAULT_GOAL := help

.PHONY: up down trace open-kibana open-apm logs logs-app clean help

up: ## Build the Spring app and start all services
	docker compose up --build -d
	@echo ""
	@echo "  Stack is starting — this takes 2-3 minutes on first run."
	@echo ""
	@echo "  Kibana   : http://localhost:5601   (elastic / $$(grep ^ELASTIC_PASSWORD .env | cut -d= -f2))"
	@echo "  APM UI   : http://localhost:5601/app/apm"
	@echo "  App      : http://localhost:8080/api/hello"
	@echo ""
	@echo "  Run 'make trace' once the app is ready to generate sample traces."

down: ## Stop and remove all containers
	docker compose down

trace: ## Send sample requests to generate traces in all four scenarios
	@echo "Sending requests to generate traces..."
	@echo ""
	@curl -sf  http://localhost:8080/api/hello                  > /dev/null && echo "  GET /api/hello            200  fast success"
	@curl -sf  "http://localhost:8080/api/slow?ms=1200"         > /dev/null && echo "  GET /api/slow?ms=1200     200  slow transaction (1.2s)"
	@curl -sf  http://localhost:8080/api/orders                 > /dev/null && echo "  GET /api/orders           200  nested spans (3 child spans)"
	@curl -sf  http://localhost:8080/api/orders/42              > /dev/null && echo "  GET /api/orders/42        200  parameterized route"
	@curl -sf  http://localhost:8080/api/error     2>/dev/null  || echo "  GET /api/error            500  error trace + stack trace in APM"
	@curl -sf  http://localhost:8080/api/orders/999 2>/dev/null || echo "  GET /api/orders/999       404  not-found error trace"
	@echo ""
	@echo "  Open Kibana → Observability → APM → spring-otel-demo"
	@echo "  http://localhost:5601/app/apm/services/spring-otel-demo/transactions"

open-kibana: ## Open Kibana in the browser
	open http://localhost:5601

open-apm: ## Open the APM service view directly
	open "http://localhost:5601/app/apm/services/spring-otel-demo/transactions"

logs: ## Follow logs for all services
	docker compose logs -f

logs-app: ## Follow Spring app logs (shows traceId/spanId per request)
	docker compose logs -f app

clean: ## Stop containers and remove all images and volumes
	docker compose down --rmi local --volumes

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-14s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
