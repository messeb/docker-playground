#!/bin/sh
echo ""
echo "🚀 App starting..."
echo ""
echo "📦 Private config fetched during build:"
echo "  ──────────────────────────────────────"
cat /etc/app/private-config.txt | sed 's/^/  /'
echo "  ──────────────────────────────────────"
echo ""
echo "🔍 Secret availability at runtime:"
if [ -n "$REGISTRY_TOKEN" ]; then
  echo "  ⚠️  REGISTRY_TOKEN env var:     $REGISTRY_TOKEN"
else
  echo "  ✅ REGISTRY_TOKEN env var:      not set"
fi

if [ -f /run/secrets/registry_token ]; then
  echo "  ⚠️  Secret file /run/secrets/:  $(cat /run/secrets/registry_token)"
else
  echo "  ✅ Secret file /run/secrets/:   not present"
fi
echo ""
