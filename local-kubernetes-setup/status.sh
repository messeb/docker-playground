#!/usr/bin/env bash
KUBECTL="kubectl --context kind-local"

SYNC=$(${KUBECTL} get application nginx-app -n argocd \
  -o jsonpath='{.status.sync.status}' 2>/dev/null || echo "unavailable")
HEALTH=$(${KUBECTL} get application nginx-app -n argocd \
  -o jsonpath='{.status.health.status}' 2>/dev/null || echo "unavailable")
DEPLOY_ID=$(${KUBECTL} get deployment myapp \
  -o jsonpath='{.spec.template.metadata.annotations.deploy-id}' 2>/dev/null || echo "unavailable")
READY=$(${KUBECTL} get deployment myapp \
  -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
DESIRED=$(${KUBECTL} get deployment myapp \
  -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
# Lines with 1 double-width emoji: string length between ┃ must be 52 (= 53 terminal cols)
# Lines without emoji: string length between ┃ must be 53
pad_var()  { printf '%*s' $(( 33 - ${#1} )) ''; }           # variable status fields (emoji prefix, 20 terminal cols)
pad_url()  { printf '%*s' $(( 33 - ${#1} )) ''; }           # url lines (emoji prefix, 20 terminal cols, url as arg)

echo ""
echo "┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓"
echo "┃                    Status                           ┃"
echo "┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫"
echo "┃  🔁 ArgoCD sync:   ${SYNC}$(pad_var "$SYNC")┃"
echo "┃  💚 ArgoCD health: ${HEALTH}$(pad_var "$HEALTH")┃"
echo "┃  📦 Deploy ID:     ${DEPLOY_ID}$(pad_var "$DEPLOY_ID")┃"
echo "┃  🚀 Pods ready:    ${READY} / ${DESIRED}$(printf '%*s' $(( 30 - ${#READY} - ${#DESIRED} )) '')┃"
echo "┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫"
echo "┃  🌐 App:           http://localhost:8888$(pad_url "http://localhost:8888")┃"
echo "┃  🔁 ArgoCD UI:     https://localhost:8080$(pad_url "https://localhost:8080")┃"
echo "┃     User: admin / argocd-local                      ┃"
echo "┃  🐙 Gitea:         http://localhost:3000$(pad_url "http://localhost:3000")┃"
echo "┃     User: gitops / gitops                           ┃"
echo "┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛"
echo ""
