#!/usr/bin/env bash
# Expose local API (port 8080) for Meta webhooks when ngrok is blocked.
# Uses Cloudflare quick tunnel (no account required).
set -euo pipefail

PORT="${PORT:-8080}"
BIN="${CLOUDFLARED_BIN:-/tmp/cloudflared}"

if [[ ! -x "$BIN" ]]; then
  echo "Downloading cloudflared..."
  curl -sfL https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -o "$BIN"
  chmod +x "$BIN"
fi

echo ""
echo "Starting tunnel to http://localhost:${PORT}"
echo ""
echo "In Meta Dashboard → Webhooks, set:"
echo "  Callback URL:  https://YOUR-SUBDOMAIN.trycloudflare.com/webhooks/meta"
echo "  Verify token:  test"
echo ""
echo "Press Ctrl+C to stop."
echo ""

exec "$BIN" tunnel --url "http://localhost:${PORT}"
