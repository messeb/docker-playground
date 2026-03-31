.DEFAULT_GOAL := help

.PHONY: up down logs ps clean generate-keys token-alice token-bob me deposit withdraw transactions create-account decrypt-token help

## ── Lifecycle ─────────────────────────────────────────────────────────────────

up: ## Build and start all services (Keycloak, PostgreSQL, API)
	docker compose up --build -d
	@echo ""
	@echo "  Services are starting. Keycloak takes ~40s on first run."
	@echo ""
	@echo "  API      : http://localhost:8081"
	@echo "  Keycloak : http://localhost:8180  (admin / admin)"
	@echo ""
	@echo "  Run 'make token-alice' once services are ready."

down: ## Stop and remove all containers
	docker compose down

logs: ## Follow logs for all services (CTRL+C to stop)
	docker compose logs -f

ps: ## Show running container status
	docker compose ps

clean: ## Stop containers and remove images + volumes (full reset)
	docker compose down --rmi local --volumes

generate-keys: ## Regenerate RSA key pair for JWE token encryption (updates .env)
	@KEY=$$(openssl genrsa 2048 2>/dev/null | base64 | tr -d '\n'); \
	 if grep -q "^API_PRIVATE_KEY_BASE64=" .env 2>/dev/null; then \
	   sed -i.bak "s|^API_PRIVATE_KEY_BASE64=.*|API_PRIVATE_KEY_BASE64=$$KEY|" .env && rm -f .env.bak; \
	 else \
	   echo "API_PRIVATE_KEY_BASE64=$$KEY" >> .env; \
	 fi; \
	 echo "New RSA key generated and written to .env — run 'make clean && make up' to apply"

## ── Demo workflow ──────────────────────────────────────────────────────────────

token-alice: ## Get a JWT for alice (role: user)  — prints full token response
	@curl -sf \
	  -X POST "http://localhost:8180/realms/banking/protocol/openid-connect/token" \
	  -d "grant_type=password&client_id=banking-api&username=alice&password=alice123" \
	  | python3 -m json.tool

token-bob: ## Get a JWT for bob (roles: user + admin) — prints full token response
	@curl -sf \
	  -X POST "http://localhost:8180/realms/banking/protocol/openid-connect/token" \
	  -d "grant_type=password&client_id=banking-api&username=bob&password=bob123" \
	  | python3 -m json.tool

me: ## Get own account info  (requires TOKEN=<access_token>)
	@curl -sf \
	  -H "Authorization: Bearer $(TOKEN)" \
	  "http://localhost:8081/api/v1/accounts/me" \
	  | python3 -m json.tool

deposit: ## Deposit 100 into own account  (requires TOKEN=<access_token>)
	@curl -sf \
	  -X POST \
	  -H "Authorization: Bearer $(TOKEN)" \
	  -H "Content-Type: application/json" \
	  -d '{"amount": 100.00}' \
	  "http://localhost:8081/api/v1/accounts/me/deposit" \
	  | python3 -m json.tool

withdraw: ## Withdraw 50 from own account  (requires TOKEN=<access_token>)
	@curl -sf \
	  -X POST \
	  -H "Authorization: Bearer $(TOKEN)" \
	  -H "Content-Type: application/json" \
	  -d '{"amount": 50.00}' \
	  "http://localhost:8081/api/v1/accounts/me/withdraw" \
	  | python3 -m json.tool

transactions: ## List transaction history  (requires TOKEN=<access_token>)
	@curl -sf \
	  -H "Authorization: Bearer $(TOKEN)" \
	  "http://localhost:8081/api/v1/accounts/me/transactions" \
	  | python3 -m json.tool

create-account: ## Create a bank account (requires TOKEN=<admin_token> ACCOUNT=<account_number> NAME=<owner_name>)
	@curl -sf \
	  -X POST \
	  -H "Authorization: Bearer $(TOKEN)" \
	  -H "Content-Type: application/json" \
	  -d "{\"account_number\": \"$(ACCOUNT)\", \"owner_name\": \"$(NAME)\"}" \
	  "http://localhost:8081/api/v1/admin/accounts" \
	  | python3 -m json.tool

decrypt-token: ## Decrypt a JWE token and display its claims  (requires TOKEN=<access_token>)
	@cd api && go run ./cmd/decode-token "$(TOKEN)"

## ── Help ──────────────────────────────────────────────────────────────────────

help: ## Show this help
	@echo ""
	@echo "Usage: make <target> [TOKEN=<jwt>]"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Quick start:"
	@echo "  make up"
	@echo "  TOKEN=\$$(curl -sf -X POST http://localhost:8180/realms/banking/protocol/openid-connect/token \\"
	@echo "    -d 'grant_type=password&client_id=banking-api&username=alice&password=alice123' \\"
	@echo "    | python3 -c \"import sys,json; print(json.load(sys.stdin)['access_token'])\")"
	@echo "  make me TOKEN=\$$TOKEN    # calls http://localhost:8081/api/v1/accounts/me"
	@echo "  make deposit TOKEN=\$$TOKEN"
	@echo ""
