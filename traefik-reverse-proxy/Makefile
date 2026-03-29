.PHONY: up down restart logs logs-service status ps open-dashboard \
        scale-a scale-b reload help

## Start all containers in the background
up:
	docker compose up -d

## Stop and remove all containers
down:
	docker compose down

## Restart all containers
restart:
	docker compose restart

## Follow logs for all containers (CTRL+C to stop)
logs:
	docker compose logs -f

## Follow logs for a specific service: make logs-service s=service-a
logs-service:
	docker compose logs -f $(s)

## Show running containers and their status
ps:
	docker compose ps

## Show Traefik routing status (routers, services, middlewares)
status:
	@python3 scripts/status.py

## Open Traefik dashboard in the browser
open-dashboard:
	open http://localhost:8080

## Scale service-a to N replicas: make scale-a n=3
scale-a:
	docker compose up -d --scale service-a=$(n) --no-recreate

## Scale service-b to N replicas: make scale-b n=3
scale-b:
	docker compose up -d --scale service-b=$(n) --no-recreate

## Reload Traefik config without downtime
reload:
	docker compose kill -s HUP traefik

## Show this help message
help:
	@grep -E '^##' Makefile | sed 's/^## //'
