#!/usr/bin/env bash
set -euo pipefail

REPO_DIR=$(cd "$(dirname "$0")" && pwd)
REPO_NAME=$(basename "$REPO_DIR")
GIT_PORT=9418

# Init git repo if needed
if [ ! -d "${REPO_DIR}/.git" ]; then
  echo "Initializing git repo..."
  git -C "${REPO_DIR}" init
  git -C "${REPO_DIR}" add .
  git -C "${REPO_DIR}" commit -m "init"
fi

# Allow git daemon to serve this repo
git -C "${REPO_DIR}" config daemon.uploadpack true
touch "${REPO_DIR}/.git/git-daemon-export-ok"

# Kill previous instances
pkill -f "git daemon.*${GIT_PORT}" 2>/dev/null || true
pkill -f "ngrok tcp ${GIT_PORT}" 2>/dev/null || true
sleep 1

# Start git daemon (log to stderr so we see errors)
echo "Starting git daemon on port ${GIT_PORT}..."
git daemon --reuseaddr \
  --base-path="${REPO_DIR}" \
  --export-all \
  --port=${GIT_PORT} \
  "${REPO_DIR}" &
GIT_DAEMON_PID=$!

# Verify git daemon is actually listening
sleep 2
if ! kill -0 "${GIT_DAEMON_PID}" 2>/dev/null; then
  echo "ERROR: git daemon failed to start."
  exit 1
fi
if ! lsof -i ":${GIT_PORT}" &>/dev/null; then
  echo "ERROR: git daemon is not listening on port ${GIT_PORT}."
  kill "${GIT_DAEMON_PID}" 2>/dev/null
  exit 1
fi
echo "git daemon running (PID ${GIT_DAEMON_PID})."

# Start ngrok TCP tunnel
echo "Starting ngrok tunnel..."
ngrok tcp ${GIT_PORT} --log=stdout --log-level=error &
NGROK_PID=$!

# Wait for ngrok API
echo "Waiting for ngrok..."
for i in $(seq 1 15); do
  NGROK_URL=$(curl -s http://localhost:4040/api/tunnels 2>/dev/null \
    | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['tunnels'][0]['public_url'])" 2>/dev/null || true)
  [ -n "${NGROK_URL}" ] && break
  sleep 1
done

if [ -z "${NGROK_URL}" ]; then
  echo "ERROR: Could not get ngrok URL. Is ngrok installed and authenticated?"
  kill "${GIT_DAEMON_PID}" "${NGROK_PID}" 2>/dev/null
  exit 1
fi

# With --base-path pointing directly at REPO_DIR, the repo is served at root "/"
REPO_URL="git://${NGROK_URL#tcp://}/"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo " Local repo is live via ngrok."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo " Repo URL: ${REPO_URL}"
echo ""
echo " Updating ArgoCD Application..."
kubectl --context kind-local patch application nginx-app -n argocd --type merge \
  -p "{\"spec\":{\"source\":{\"repoURL\":\"${REPO_URL}\"}}}" \
  && echo " ArgoCD updated." \
  || echo " Could not patch ArgoCD (is the cluster running?). Set repoURL manually."
kubectl --context kind-local annotate app nginx-app -n argocd \
  argocd.argoproj.io/refresh=hard --overwrite &>/dev/null \
  && echo " ArgoCD sync triggered." || true
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Press Ctrl+C to stop."

trap "kill ${GIT_DAEMON_PID} ${NGROK_PID} 2>/dev/null; echo 'Stopped.'" INT TERM
wait
