REGISTRY   = localhost:5001
IMAGE_NAME = myapp
IMAGE_TAG  = latest
IMAGE      = $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: build deploy serve-local help

help:
	@echo "Usage:"
	@echo "  make build        Build image and push to local registry"
	@echo "  make deploy       Release a new version end-to-end"
	@echo "  make serve-local  Serve this repo via ngrok and update ArgoCD"

build:
	docker build -t $(IMAGE) .
	docker push $(IMAGE)

deploy:
	$(eval BUILD_ID  := $(shell openssl rand -hex 4))
	$(eval TIMESTAMP := $(shell date '+%Y-%m-%d %H:%M:%S'))
	@echo ""
	@echo "🚀 Starting release..."
	@echo "   📦 Build:  $(BUILD_ID)"
	@echo "   🕐 Date:   $(TIMESTAMP)"
	@echo ""
	@docker build \
		--build-arg BUILD_ID="$(BUILD_ID)" \
		--build-arg BUILD_DATE="$(TIMESTAMP)" \
		-t $(IMAGE) . -q >/dev/null
	@echo "   ✅ Docker image built"
	@docker push $(IMAGE) -q >/dev/null
	@echo "   ✅ Image pushed → $(IMAGE)"
	@sed -i '' 's/deploy-id: ".*"/deploy-id: "$(BUILD_ID)"/' apps/deployment.yaml
	@echo "   ✅ deployment.yaml patched"
	@git add apps/deployment.yaml
	@git commit -m "🚀 deploy: $(BUILD_ID)" --quiet
	@echo "   ✅ Committed"
	@git push http://gitops:gitops@localhost:3000/gitops/app.git HEAD:main --force --quiet 2>/dev/null
	@echo "   ✅ Pushed to Gitea"
	@kubectl --context kind-local annotate app nginx-app -n argocd \
		argocd.argoproj.io/refresh=hard --overwrite --quiet 2>/dev/null || true
	@echo "   ✅ ArgoCD sync triggered"
	@echo ""
	@echo "✨ Release $(BUILD_ID) live in seconds"
	@echo ""
	@echo "   🌐 http://localhost:8888"
	@echo ""

serve-local:
	-./serve-local.sh
