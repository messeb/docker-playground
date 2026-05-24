EXAMPLES := service-invocation state-management pubsub bindings-secrets

.PHONY: help up-all down-all clean

help:
	@echo "Targets:"
	@echo "  up-<example>     start: $(EXAMPLES)"
	@echo "  down-<example>   stop + remove volumes"
	@echo "  logs-<example>   follow logs"
	@echo "  ps-<example>     status"
	@echo "  test-<example>   curl demo endpoint"
	@echo "  up-all / down-all / clean"

up-%:
	docker compose -f $*/docker-compose.yml up -d --build

down-%:
	docker compose -f $*/docker-compose.yml down -v

logs-%:
	docker compose -f $*/docker-compose.yml logs -f

ps-%:
	docker compose -f $*/docker-compose.yml ps

up-all: $(addprefix up-,$(EXAMPLES))
down-all: $(addprefix down-,$(EXAMPLES))

test-service-invocation:
	@curl -s localhost:8001/call/world | tee /dev/stderr ; echo

test-state-management:
	@curl -s -X POST localhost:8001/save -H 'content-type: application/json' -d '{"key":"k1","value":"hello"}' | tee /dev/stderr ; echo
	@sleep 1
	@curl -s localhost:8002/load/k1 | tee /dev/stderr ; echo

test-pubsub:
	@curl -s -X POST localhost:8001/publish -H 'content-type: application/json' -d '{"message":"order-1"}' | tee /dev/stderr ; echo
	@echo "Check subscriber logs: make logs-pubsub"

test-bindings-secrets:
	@curl -s localhost:8001/info | tee /dev/stderr ; echo
	@echo "Cron ticks: make logs-bindings-secrets"

clean: down-all
	docker system prune -f
