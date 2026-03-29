#!/usr/bin/env bash
set -euo pipefail

# Detect container engine (respect CONTAINER_ENGINE env var if set)
if [ -z "${CONTAINER_ENGINE:-}" ]; then
  if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
    CONTAINER_ENGINE="docker"
  elif command -v podman &>/dev/null; then
    CONTAINER_ENGINE="podman"
  else
    CONTAINER_ENGINE="docker"
  fi
fi
if [ "${CONTAINER_ENGINE}" = "podman" ]; then
  export KIND_EXPERIMENTAL_PROVIDER=podman
fi

kind delete cluster --name local
${CONTAINER_ENGINE} rm -f kind-registry 2>/dev/null || true

# Reset app folder to a clean state
rm -rf app/.git
git -C app checkout -- apps/deployment.yaml 2>/dev/null || true

echo "Done. Cluster, registry, and app folder reset."
