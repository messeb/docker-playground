#!/usr/bin/env bash
set -euo pipefail

ARGOCD_VERSION="v3.0.0"
CLUSTER_NAME="local"
ARGOCD_NAMESPACE="argocd"
KUBECTL="kubectl --context kind-${CLUSTER_NAME}"

# Detect container engine (respect CONTAINER_ENGINE env var if set)
if [ -z "${CONTAINER_ENGINE:-}" ]; then
  if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
    CONTAINER_ENGINE="docker"
  elif command -v podman &>/dev/null; then
    CONTAINER_ENGINE="podman"
  else
    echo "ERROR: Neither docker nor podman found or running." >&2
    exit 1
  fi
fi
if [ "${CONTAINER_ENGINE}" = "podman" ]; then
  export KIND_EXPERIMENTAL_PROVIDER=podman
fi
echo "Using container engine: ${CONTAINER_ENGINE}"

# Start local registry (idempotent)
if ${CONTAINER_ENGINE} inspect kind-registry &>/dev/null; then
  echo "Local registry already running — skipping."
else
  echo "Starting local registry on port 5001..."
  ${CONTAINER_ENGINE} run -d \
    --name kind-registry \
    --restart=always \
    -p 5001:5000 \
    registry:2
fi

# Create kind cluster (idempotent)
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  echo "kind cluster '${CLUSTER_NAME}' already exists — skipping."
else
  echo "Creating kind cluster '${CLUSTER_NAME}'..."
  kind create cluster --config cluster/kind-config.yaml
fi

# Merge kind kubeconfig into ~/.kube/config
kind export kubeconfig --name "${CLUSTER_NAME}"

# Connect registry to kind network (idempotent)
if ${CONTAINER_ENGINE} network inspect kind | grep -q "kind-registry"; then
  echo "Registry already connected to kind network."
else
  echo "Connecting registry to kind network..."
  ${CONTAINER_ENGINE} network connect kind kind-registry
fi

# Install ArgoCD (idempotent)
if ${KUBECTL} get namespace "${ARGOCD_NAMESPACE}" &>/dev/null; then
  echo "ArgoCD already installed — skipping."
else
  echo "Installing ArgoCD ${ARGOCD_VERSION}..."
  ${KUBECTL} create namespace "${ARGOCD_NAMESPACE}"
  ${KUBECTL} apply -n "${ARGOCD_NAMESPACE}" \
    -f "https://raw.githubusercontent.com/argoproj/argo-cd/${ARGOCD_VERSION}/manifests/install.yaml" \
    &>/dev/null
  echo "ArgoCD resources created."
fi

# Set ArgoCD repo polling interval to 1 minute
${KUBECTL} -n "${ARGOCD_NAMESPACE}" patch configmap argocd-cm \
  --type merge -p '{"data":{"timeout.reconciliation":"60s"}}' &>/dev/null

# Wait for ArgoCD server (also ensures CRDs are registered)
echo "Waiting for ArgoCD pods to be ready (this takes ~1-2 min)..."
until ${KUBECTL} -n "${ARGOCD_NAMESPACE}" get pods 2>/dev/null | grep -q "argocd-server"; do
  sleep 2
done
${KUBECTL} rollout status deployment/argocd-server -n "${ARGOCD_NAMESPACE}" --timeout=180s

# Deploy Gitea git server
if ${KUBECTL} get deployment gitea &>/dev/null; then
  echo "Gitea already deployed — skipping."
else
  echo "Deploying Gitea..."
  ${KUBECTL} apply -f gitea/ &>/dev/null
  ${KUBECTL} rollout status deployment/gitea --timeout=120s
fi

# Wait for Gitea HTTP to be ready
echo "Waiting for Gitea to be accessible..."
until curl -s http://localhost:3000 >/dev/null 2>&1; do sleep 2; done

# Create admin user (idempotent)
${KUBECTL} exec deployment/gitea -- gitea admin user create \
  --username gitops \
  --password gitops \
  --email gitops@local.dev \
  --admin \
  --must-change-password=false 2>/dev/null || true

# Create app repo (idempotent)
curl -s -X POST http://localhost:3000/api/v1/user/repos \
  -u gitops:gitops \
  -H "Content-Type: application/json" \
  -d '{"name":"app","private":false,"default_branch":"main","auto_init":false}' \
  >/dev/null || true

# Initialize app git repo and push to Gitea
GITEA_REMOTE="http://gitops:gitops@localhost:3000/gitops/app.git"
if [ ! -d "app/.git" ]; then
  echo "Initializing app git repo..."
  git -C app init -b main
  git -C app -c user.email="gitops@local" -c user.name="gitops" add .
  git -C app -c user.email="gitops@local" -c user.name="gitops" commit -m "init"
fi
git -C app remote set-url gitea "${GITEA_REMOTE}" 2>/dev/null \
  || git -C app remote add gitea "${GITEA_REMOTE}"
git -C app push gitea HEAD:main --force --quiet
echo "App pushed to Gitea."

# Apply ArgoCD Application
echo "Applying ArgoCD Application..."
${KUBECTL} apply -f argocd/application.yaml

# Start ArgoCD port-forward
${KUBECTL} port-forward svc/argocd-server -n argocd 8080:443 &>/dev/null &

# Set ArgoCD admin password to "argocd"
echo "Setting ArgoCD admin password..."
until curl -sk https://localhost:8080/api/v1/session >/dev/null 2>&1; do sleep 1; done
INITIAL_PASSWORD=$(${KUBECTL} -n "${ARGOCD_NAMESPACE}" get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d)
TOKEN=$(curl -sk -X POST https://localhost:8080/api/v1/session \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"admin\",\"password\":\"${INITIAL_PASSWORD}\"}" \
  | python3 -c "import sys,json; print(json.load(sys.stdin).get('token',''))")
curl -sk -X PUT https://localhost:8080/api/v1/account/password \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"currentPassword\":\"${INITIAL_PASSWORD}\",\"newPassword\":\"argocd-local\"}" >/dev/null

echo ""
echo "┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓"
echo "┃           ✅  Cluster ready!                        ┃"
echo "┃              Running initial deploy...              ┃"
echo "┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫"
echo "┃  🌐 App:      http://localhost:8888                 ┃"
echo "┃  🔁 ArgoCD:   https://localhost:8080                ┃"
echo "┃     User: admin / Pass: argocd-local                ┃"
echo "┃  🐙 Gitea:    http://localhost:3000                 ┃"
echo "┃     User: gitops / Pass: gitops                     ┃"
echo "┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛"
echo ""
