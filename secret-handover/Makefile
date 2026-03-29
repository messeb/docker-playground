REGISTRY_IMAGE     := private-registry
REGISTRY_CONTAINER := private-registry
REGISTRY_PORT      := 19080
TOKEN_FILE         := secrets/registry_token.txt

.PHONY: all demo build-secure build-insecure verify run-secure run-insecure \
        registry clean help

# ── Help ──────────────────────────────────────────────────────────────────────

help:
	@echo ""
	@echo "Usage:"
	@echo "  make demo            Full demo: build both images, verify, run both"
	@echo "  make build-secure    Build image using --mount=type=secret (safe)"
	@echo "  make build-insecure  Build image using --build-arg (leaks token)"
	@echo "  make verify          Check whether the token appears in image layers"
	@echo "  make run-secure      Run the secure app container"
	@echo "  make run-insecure    Run the insecure app container"
	@echo "  make clean           Remove containers and images"
	@echo ""

# ── Infrastructure ────────────────────────────────────────────────────────────

registry:
	@if docker ps -q -f name=^/$(REGISTRY_CONTAINER)$$ | grep -q .; then \
	  echo "  [registry] already running on port $(REGISTRY_PORT)"; \
	else \
	  docker rm -f $(REGISTRY_CONTAINER) 2>/dev/null || true; \
	  docker build -q -t $(REGISTRY_IMAGE) ./private-registry; \
	  docker run -d \
	    --name $(REGISTRY_CONTAINER) \
	    -p $(REGISTRY_PORT):8080 \
	    -e REGISTRY_TOKEN=$$(cat $(TOKEN_FILE)) \
	    $(REGISTRY_IMAGE) > /dev/null; \
	  sleep 1; \
	  echo "  [registry] started on port $(REGISTRY_PORT)"; \
	fi

# ── Build targets ─────────────────────────────────────────────────────────────

build-secure: registry
	@echo ""
	@echo "Building SECURE image (--mount=type=secret) ..."
	@DOCKER_BUILDKIT=1 docker build \
	  --secret id=registry_token,src=$(TOKEN_FILE) \
	  -t app-secure \
	  -f Dockerfile .
	@echo "  [build-secure] done"

build-insecure: registry
	@echo ""
	@echo "Building INSECURE image (--build-arg) ..."
	@DOCKER_BUILDKIT=1 docker build \
	  --build-arg REGISTRY_TOKEN=$$(cat $(TOKEN_FILE)) \
	  -t app-insecure \
	  -f Dockerfile.insecure .
	@echo "  [build-insecure] done"

# ── Verify ────────────────────────────────────────────────────────────────────

verify:
	@echo ""
	@echo "========================================================"
	@echo "  Layer inspection: is the token visible in history?"
	@echo "========================================================"
	@TOKEN=$$(cat $(TOKEN_FILE) | tr -d '[:space:]'); \
	echo ""; \
	echo "  Token: $$TOKEN"; \
	echo ""; \
	if docker history --no-trunc app-secure 2>/dev/null | grep -qF "$$TOKEN"; then \
	  echo "  [app-secure]    FAIL — token found in layer history!"; \
	else \
	  echo "  [app-secure]    PASS — token NOT found in layer history"; \
	fi; \
	if docker history --no-trunc app-insecure 2>/dev/null | grep -qF "$$TOKEN"; then \
	  echo "  [app-insecure]  FAIL — token found in layer history (as expected)"; \
	else \
	  echo "  [app-insecure]  PASS — token not found (unexpected)"; \
	fi
	@echo ""

# ── Run ───────────────────────────────────────────────────────────────────────

run-secure:
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  🔒  SECURE IMAGE  (app-secure)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@docker run --rm app-secure
	@echo "🔎 Layer history scan:"
	@if docker history --no-trunc app-secure | grep -qF "$$(cat $(TOKEN_FILE) | tr -d '[:space:]')"; then \
	  echo "  ❌ Token found in image history!"; \
	else \
	  echo "  ✅ Token NOT found — secret never written to any layer"; \
	fi
	@echo ""

run-insecure:
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  ⚠️  INSECURE IMAGE  (app-insecure)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@docker run --rm app-insecure
	@echo "🔎 Layer history scan:"
	@docker history --no-trunc app-insecure \
	  | grep -o 'REGISTRY_TOKEN=[^ ]*' \
	  | head -1 \
	  | sed 's/REGISTRY_TOKEN=/  ❌ Token exposed in history: /'
	@echo ""

# ── Full demo ─────────────────────────────────────────────────────────────────

demo: build-secure build-insecure verify run-secure run-insecure
	@echo ""
	@echo "Demo complete."
	@echo "To inspect layers manually:"
	@echo "  docker history --no-trunc app-insecure | grep curl"
	@echo ""

# ── Cleanup ───────────────────────────────────────────────────────────────────

clean:
	@docker rm -f $(REGISTRY_CONTAINER) 2>/dev/null || true
	@docker rmi app-secure app-insecure $(REGISTRY_IMAGE) 2>/dev/null || true
	@echo "Cleaned up."
