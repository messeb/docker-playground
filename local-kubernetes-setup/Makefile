.PHONY: start deploy teardown status deps help

help:
	@echo "Usage:"
	@echo "  make start      Boot the cluster and set up everything (one command)"
	@echo "  make deploy     Build, push, and release a new app version"
	@echo "  make status     Show sync state, health, deploy ID, and pod readiness"
	@echo "  make teardown   Tear down the cluster, registry, and reset app"
	@echo ""
	@echo "Utilities:"
	@echo "  make deps       Install kind + kubectl via Homebrew"

deps:
	@echo "Installing dependencies..."
	@command -v brew >/dev/null 2>&1 || { echo "ERROR: Homebrew not found. Install it from https://brew.sh"; exit 1; }
	@(command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1) || \
	  command -v podman >/dev/null 2>&1 || \
	  { echo "ERROR: Neither Docker nor Podman found. Install Docker Desktop or Podman first."; exit 1; }
	@command -v kind >/dev/null 2>&1 && echo "kind already installed." || brew install kind
	@command -v kubectl >/dev/null 2>&1 && echo "kubectl already installed." || brew install kubectl
	@echo "All dependencies ready."

start: deps
	./start.sh
	$(MAKE) deploy

deploy:
	$(MAKE) -C app deploy

status:
	@./status.sh

teardown:
	./teardown.sh
