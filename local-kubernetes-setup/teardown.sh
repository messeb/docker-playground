#!/usr/bin/env bash
set -euo pipefail
kind delete cluster --name local
docker rm -f kind-registry 2>/dev/null || true

# Reset app folder to a clean state
rm -rf app/.git
git -C app checkout -- apps/deployment.yaml 2>/dev/null || true

echo "Done. Cluster, registry, and app folder reset."
