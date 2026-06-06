.DEFAULT_GOAL := help
SHELL         := /bin/bash

# ── Container runtime auto-detect ────────────────────────────────────────────
# Tries docker first, falls back to podman. Override by exporting:
#   RUNTIME=podman COMPOSE='podman compose'
RUNTIME ?= $(shell \
  if command -v docker  >/dev/null 2>&1; then echo docker;  \
  elif command -v podman >/dev/null 2>&1; then echo podman; \
  fi)

COMPOSE ?= $(shell \
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then \
    echo "docker compose"; \
  elif command -v podman-compose >/dev/null 2>&1; then \
    echo "podman-compose"; \
  elif command -v podman >/dev/null 2>&1 && podman compose version >/dev/null 2>&1; then \
    echo "podman compose"; \
  fi)

ifeq ($(strip $(RUNTIME)),)
  $(error No container runtime found. Install Docker (https://docs.docker.com/get-docker/) or Podman (https://podman.io/docs/installation))
endif
ifeq ($(strip $(COMPOSE)),)
  $(error No compose plugin found. Install 'docker compose' or 'podman-compose')
endif

# ── ANSI helpers ─────────────────────────────────────────────────────────────
C_RESET := \033[0m
C_BOLD  := \033[1m
C_DIM   := \033[2m
C_RED   := \033[31m
C_GREEN := \033[32m
C_YEL   := \033[33m
C_BLUE  := \033[34m
C_MAG   := \033[35m
C_CYAN  := \033[36m

.PHONY: help up down restart rebuild logs ps clean status-services \
        generate-keys doctor urls reset-carol \
        login mail kc \
        status me deposit withdraw transactions \
        forge-valid forge-tampered

## ────────────────────────────────────────────────────────────────────────────
## Lifecycle
## ────────────────────────────────────────────────────────────────────────────

up: ## Build images and start every service (one shot, idempotent)
	@if [ ! -f .env ]; then cp .env.example .env; fi
	@if ! grep -q "^AUTHZ_PRIVATE_KEY_BASE64=." .env || \
	    ! grep -q "^API_PRIVATE_KEY_BASE64=." .env || \
	    ! grep -q "^WRAPPER_HMAC_SECRET=." .env; then \
	   printf "$(C_YEL)  → secrets missing in .env, generating…$(C_RESET)\n"; \
	   $(MAKE) -s generate-keys; \
	fi
	@$(COMPOSE) up --build -d
	@printf "\n  $(C_BOLD)Services starting.$(C_RESET) Keycloak takes ~40s on first run.\n\n"
	@$(MAKE) -s urls
	@printf "\n  Sign in at $(C_CYAN)http://localhost:8080$(C_RESET) — try $(C_BOLD)alice / alice123$(C_RESET) (PIN appears in Mailpit).\n"

down: ## Stop and remove all containers (keeps volumes)
	@$(COMPOSE) down

restart: ## docker compose down + up --build (no volume wipe)
	@$(COMPOSE) down
	@$(MAKE) -s up

rebuild: ## Rebuild a single service quickly, e.g. `make rebuild SERVICE=api`
	@if [ -z "$(SERVICE)" ]; then \
	   printf "$(C_RED)  ERR: pass SERVICE=<name>$(C_RESET)\n"; exit 1; \
	fi
	@$(COMPOSE) up --build -d $(SERVICE)

logs: ## Follow logs from all services (CTRL+C to stop). Filter via SERVICE=
	@$(COMPOSE) logs -f $(SERVICE)

ps: ## Container status
	@$(COMPOSE) ps

clean: ## Stop containers + remove built images + volumes (full wipe)
	@$(COMPOSE) down --rmi local --volumes

## ────────────────────────────────────────────────────────────────────────────
## Setup
## ────────────────────────────────────────────────────────────────────────────

doctor: ## Check that all prerequisites are present
	@printf "$(C_BOLD)environment$(C_RESET)\n"
	@printf "  runtime    : $(C_CYAN)$(RUNTIME)$(C_RESET)\n"
	@printf "  compose    : $(C_CYAN)$(COMPOSE)$(C_RESET)\n"
	@for cmd in openssl curl awk python3; do \
	   if command -v $$cmd >/dev/null 2>&1; then \
	     printf "  %-10s : $(C_GREEN)✓$(C_RESET) %s\n" "$$cmd" "$$($$cmd --version 2>/dev/null | head -1 || echo present)"; \
	   else \
	     printf "  %-10s : $(C_RED)✗ missing$(C_RESET)\n" "$$cmd"; \
	   fi; \
	 done
	@printf "\n$(C_BOLD)host ports$(C_RESET) (must be free)\n"
	@for p in 8080 8180 8025 5432; do \
	   if lsof -nP -iTCP:$$p -sTCP:LISTEN >/dev/null 2>&1; then \
	     printf "  %-6s : $(C_RED)✗ in use$(C_RESET)\n" "$$p"; \
	   else \
	     printf "  %-6s : $(C_GREEN)✓ free$(C_RESET)\n" "$$p"; \
	   fi; \
	 done

generate-keys: ## (Re)generate RSA keypairs + HMAC secret in .env
	@if [ ! -f .env ]; then cp .env.example .env; fi
	@for VAR in AUTHZ_PRIVATE_KEY_BASE64 API_PRIVATE_KEY_BASE64; do \
	   KEY=$$(openssl genrsa 2048 2>/dev/null | base64 | tr -d '\n'); \
	   if grep -q "^$$VAR=" .env; then \
	     sed -i.bak "s|^$$VAR=.*|$$VAR=$$KEY|" .env && rm -f .env.bak; \
	   else \
	     printf "%s=%s\n" "$$VAR" "$$KEY" >> .env; \
	   fi; \
	   printf "  $(C_GREEN)✓$(C_RESET) %s written\n" "$$VAR"; \
	 done
	@HMAC=$$(openssl rand -base64 32 | tr -d '\n'); \
	 if grep -q "^WRAPPER_HMAC_SECRET=" .env; then \
	   sed -i.bak "s|^WRAPPER_HMAC_SECRET=.*|WRAPPER_HMAC_SECRET=$$HMAC|" .env && rm -f .env.bak; \
	 else \
	   printf "WRAPPER_HMAC_SECRET=%s\n" "$$HMAC" >> .env; \
	 fi; \
	 printf "  $(C_GREEN)✓$(C_RESET) WRAPPER_HMAC_SECRET written\n"

urls: ## Print every published URL the demo exposes
	@printf "  $(C_BOLD)SPA / API gateway$(C_RESET)   $(C_CYAN)http://localhost:8080$(C_RESET)\n"
	@printf "  $(C_BOLD)Keycloak admin$(C_RESET)      $(C_CYAN)http://localhost:8180/admin$(C_RESET)    (admin / admin)\n"
	@printf "  $(C_BOLD)Mailpit (OTP)$(C_RESET)       $(C_CYAN)http://localhost:8025$(C_RESET)\n"

reset-carol: ## Reset carol back to no-MFA so the modal flow can be demoed again
	@$(COMPOSE) exec -T keycloak sh -c '\
	  /opt/keycloak/bin/kcadm.sh config credentials \
	    --server http://localhost:8080 --realm master \
	    --user $${KEYCLOAK_ADMIN:-$${KEYCLOAK_ADMIN_USER:-admin}} \
	    --password $${KEYCLOAK_ADMIN_PASSWORD:-admin} >/dev/null && \
	  ID=$$(/opt/keycloak/bin/kcadm.sh get users -r banking-mfa -q username=carol --fields id 2>/dev/null | grep -oE "[0-9a-f-]{36}") && \
	  /opt/keycloak/bin/kcadm.sh update users/$$ID -r banking-mfa \
	    -s "email=carol@example.com" \
	    -s "emailVerified=true" \
	    -s "attributes.bank_account_number=[\"BANK-0003-2024\"]" \
	    -s "attributes.mfa_enabled=" \
	    -s "attributes.mfa_method="' >/dev/null
	@$(COMPOSE) restart api >/dev/null
	@printf "  $(C_GREEN)✓$(C_RESET) carol Keycloak attributes wiped\n"
	@printf "  $(C_GREEN)✓$(C_RESET) API restarted (in-memory step-up session cleared)\n"

## ────────────────────────────────────────────────────────────────────────────
## Browser shortcuts (cross-platform)
## ────────────────────────────────────────────────────────────────────────────

login: ## Open the SPA login page
	@$(call _open,http://localhost:8080)

mail: ## Open Mailpit (catches MFA codes)
	@$(call _open,http://localhost:8025)

kc: ## Open Keycloak admin console
	@$(call _open,http://localhost:8180/admin)

# Cross-platform open helper.
define _open
   open "$1" 2>/dev/null || xdg-open "$1" 2>/dev/null || start "$1" 2>/dev/null || echo "Open $1"
endef

## ────────────────────────────────────────────────────────────────────────────
## Demo calls (need TOKEN= the JWE from sessionStorage.access_token)
## ────────────────────────────────────────────────────────────────────────────

status: ## GET /api/v1/mfa/status (requires TOKEN=…)
	@$(call _api_get,/api/v1/mfa/status)

me: ## GET /api/v1/accounts/me
	@$(call _api_get,/api/v1/accounts/me)

transactions: ## GET /api/v1/accounts/me/transactions
	@$(call _api_get,/api/v1/accounts/me/transactions)

deposit: ## POST /api/v1/accounts/me/deposit · 100
	@$(call _api_post,/api/v1/accounts/me/deposit,{"amount":100})

withdraw: ## POST /api/v1/accounts/me/withdraw · 50
	@$(call _api_post,/api/v1/accounts/me/withdraw,{"amount":50})

define _check_token
   if [ -z "$(TOKEN)" ]; then \
     printf "$(C_RED)  ERR: pass TOKEN=… (copy from devtools: sessionStorage.getItem('access_token'))$(C_RESET)\n"; \
     exit 1; \
   fi
endef

define _api_get
   @$(_check_token); \
   curl -sf -H "Authorization: Bearer $(TOKEN)" \
        "http://localhost:8080$1" | python3 -m json.tool
endef

define _api_post
   @$(_check_token); \
   curl -sf -X POST -H "Authorization: Bearer $(TOKEN)" -H "Content-Type: application/json" \
        -d '$2' "http://localhost:8080$1" | python3 -m json.tool
endef

## ────────────────────────────────────────────────────────────────────────────
## HMAC trust demonstration — hit api:8080 directly from inside the network
## ────────────────────────────────────────────────────────────────────────────
##   make forge-valid    USER=alice            → expect 200
##   make forge-tampered USER=alice TAMPER=X-User-Roles=admin → expect 401

USER  ?= alice
PATH_ ?= /api/v1/accounts/me

ifeq ($(USER),alice)
  _SUB     := aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee
  _EMAIL   := alice@example.com
  _ROLES   := user
  _ACCOUNT := BANK-0001-2024
endif
ifeq ($(USER),bob)
  _SUB     := 11111111-2222-3333-4444-555555555555
  _EMAIL   := bob@example.com
  _ROLES   := user
  _ACCOUNT := BANK-0002-2024
endif
ifeq ($(USER),carol)
  _SUB     := cccccccc-dddd-eeee-ffff-aaaaaaaaaaaa
  _EMAIL   := carol@example.com
  _ROLES   := user
  _ACCOUNT := BANK-0003-2024
endif

forge-valid: ## Hit api:8080 directly with a valid HMAC-signed header set (expect 200)
	@$(call _forge,)

forge-tampered: ## Same signature, but mutate one header AFTER signing (expect 401)
	@if [ -z "$(TAMPER)" ]; then \
	   printf "$(C_RED)  ERR: pass TAMPER=<Header>=<value>, e.g. TAMPER=X-User-Roles=admin$(C_RESET)\n"; \
	   exit 1; \
	 fi
	@$(call _forge,$(TAMPER))

# $1 (optional) = "Header=value" to mutate after signing.
define _forge
   SECRET=$$(awk -F= '/^WRAPPER_HMAC_SECRET=/ { sub(/^[^=]*=/, ""); print }' .env); \
   TS=$$(date +%s); \
   USERNAME=$(USER); SUB=$(_SUB); EMAIL=$(_EMAIL); ROLES=$(_ROLES); ACC=$(_ACCOUNT); \
   CANON=$$(printf 'x-bank-account-number=%s\nx-mfa-method=email\nx-mfa-verified=true\nx-user-email=%s\nx-user-roles=%s\nx-user-sub=%s\nx-user-username=%s\nx-wrapper-timestamp=%s' \
       "$$ACC" "$$EMAIL" "$$ROLES" "$$SUB" "$$USERNAME" "$$TS"); \
   SIG=$$(printf '%s' "$$CANON" | openssl dgst -sha256 -hmac "$$SECRET" -binary \
       | openssl base64 | tr '+/' '-_' | tr -d '=\n'); \
   if [ -n "$1" ]; then \
     TKEY=$$(echo "$1" | cut -d= -f1); TVAL=$$(echo "$1" | cut -d= -f2-); \
     case "$$TKEY" in \
       X-User-Sub) SUB="$$TVAL" ;; X-User-Username) USERNAME="$$TVAL" ;; \
       X-User-Email) EMAIL="$$TVAL" ;; X-User-Roles) ROLES="$$TVAL" ;; \
       X-Bank-Account-Number) ACC="$$TVAL" ;; \
     esac; \
     printf "  $(C_YEL)→$(C_RESET) tampered %s = %s  (sig was computed first)\n" "$$TKEY" "$$TVAL"; \
   else \
     printf "  $(C_GREEN)→$(C_RESET) %s ($(PATH_))  ts=%s  sig=%s…\n" "$$USERNAME" "$$TS" "$${SIG:0:16}"; \
   fi; \
   $(RUNTIME) run --rm --network bank-mfa-net curlimages/curl:latest \
       curl -sS -o - -w "\n  HTTP %{http_code}\n" \
       -H "X-User-Sub: $$SUB" \
       -H "X-User-Username: $$USERNAME" \
       -H "X-User-Email: $$EMAIL" \
       -H "X-User-Roles: $$ROLES" \
       -H "X-Bank-Account-Number: $$ACC" \
       -H "X-MFA-Verified: true" \
       -H "X-MFA-Method: email" \
       -H "X-Wrapper-Timestamp: $$TS" \
       -H "X-Wrapper-Signature: $$SIG" \
       "http://api:8080$(PATH_)"
endef

## ────────────────────────────────────────────────────────────────────────────
## Help
## ────────────────────────────────────────────────────────────────────────────

help: ## Show this help
	@printf "\n  $(C_BOLD)Banking · MFA demo$(C_RESET)  $(C_DIM)— runtime: $(RUNTIME), compose: $(COMPOSE)$(C_RESET)\n\n"
	@awk 'BEGIN { FS = ":.*##"; section = "" } \
	  /^## ───/ { in_section = 1; next } \
	  /^## [A-Za-z]/ && in_section { \
	    sub(/^## /, ""); printf "\n  $(C_BOLD)%s$(C_RESET)\n", $$0; in_section = 0; next } \
	  /^[a-zA-Z_-]+:.*##/ { printf "    $(C_CYAN)%-16s$(C_RESET) %s\n", $$1, $$2 }' \
	  $(MAKEFILE_LIST)
	@printf "\n  $(C_BOLD)Quickstart$(C_RESET)\n"
	@printf "    make doctor    → verify prerequisites\n"
	@printf "    make up        → build images, start everything, generate keys if needed\n"
	@printf "    make login     → open http://localhost:8080 (alice/alice123 or bob/bob123 or carol/carol123)\n"
	@printf "    make urls      → list every published endpoint\n\n"
