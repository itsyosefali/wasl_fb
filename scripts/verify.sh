#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-demo-api-key-change-in-production}"
META_SECRET="${META_APP_SECRET:-your-meta-app-secret}"

echo "==> Health check"
curl -sf "$API_URL/health" | grep -q '"status":"ok"' && echo "OK"

echo "==> Connect page (skip if already connected)"
PAGES=$(curl -sf -H "X-API-Key: $API_KEY" "$API_URL/pages")
if echo "$PAGES" | grep -q '"meta_page_id":"123456789"'; then
  echo "Page already connected"
else
  curl -sf -X POST "$API_URL/pages/connect" \
    -H "X-API-Key: $API_KEY" \
    -H "Content-Type: application/json" \
    -d '{"meta_page_id":"123456789","name":"Verify Page","access_token":"test-token"}' > /dev/null
fi

echo "==> Register webhook (skip if already registered)"
HOOKS=$(curl -sf -H "X-API-Key: $API_KEY" "$API_URL/webhooks")
if echo "$HOOKS" | grep -q 'webhook-catcher'; then
  echo "Webhook already registered"
else
  curl -sf -X POST "$API_URL/webhooks" \
    -H "X-API-Key: $API_KEY" \
    -H "Content-Type: application/json" \
    -d '{"url":"http://webhook-catcher:8080/webhook"}' > /dev/null
fi

echo "==> Simulate signed Meta message webhook"
PAYLOAD='{"object":"page","entry":[{"id":"123456789","messaging":[{"sender":{"id":"verify_user"},"recipient":{"id":"123456789"},"timestamp":1716979200,"message":{"mid":"m_verify","text":"Hello from verify script"}}]}]}'
SIG=$(printf '%s' "$PAYLOAD" | openssl dgst -sha256 -hmac "$META_SECRET" | sed 's/^.* //')
RESULT=$(curl -sf -X POST "$API_URL/webhooks/meta" \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: sha256=$SIG" \
  -d "$PAYLOAD")
echo "$RESULT" | grep -q '"processed":1' && echo "Ingest OK"

sleep 2

echo "==> Verify message persisted"
curl -sf -H "X-API-Key: $API_KEY" "$API_URL/messages" | grep -q 'm_verify' && echo "Message OK"

echo "==> Verify Meta hub challenge"
CHALLENGE=$(curl -sf "$API_URL/webhooks/meta?hub.mode=subscribe&hub.verify_token=your-meta-verify-token&hub.challenge=verify123")
[ "$CHALLENGE" = "verify123" ] && echo "Challenge OK"

echo ""
echo "All verification checks passed."
