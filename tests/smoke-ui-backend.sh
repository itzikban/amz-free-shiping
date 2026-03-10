#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BACKEND_DIR="$ROOT/backend"
FRONTEND_DIR="$ROOT/frontend"

BACK_PID=""
FRONT_PID=""
cleanup() {
  [[ -n "$BACK_PID" ]] && kill "$BACK_PID" 2>/dev/null || true
  [[ -n "$FRONT_PID" ]] && kill "$FRONT_PID" 2>/dev/null || true
}
trap cleanup EXIT

export DECODO_BASIC_AUTH="${DECODO_BASIC_AUTH:-}"
if [[ -z "$DECODO_BASIC_AUTH" ]]; then
  echo "[warn] DECODO_BASIC_AUTH not set; live shipping check may fail or be less accurate"
fi

# free ports if occupied by stale processes
fuser -k 8085/tcp 2>/dev/null || true
fuser -k 8002/tcp 2>/dev/null || true

(
  cd "$BACKEND_DIR"
  go run ./cmd/server >/tmp/amz-backend.log 2>&1
) &
BACK_PID=$!

for i in {1..30}; do
  if curl -fsS http://127.0.0.1:8085/health >/dev/null; then
    break
  fi
  sleep 1
done
curl -fsS http://127.0.0.1:8085/health >/dev/null

echo "[ok] backend health"

(
  cd "$FRONTEND_DIR"
  BACKEND_BASE_URL="http://127.0.0.1:8085" npm run dev -- -p 8002 >/tmp/amz-frontend.log 2>&1
) &
FRONT_PID=$!

for i in {1..40}; do
  if curl -fsS http://127.0.0.1:8002/api/health >/dev/null; then
    break
  fi
  sleep 1
done
curl -fsS http://127.0.0.1:8002/api/health >/dev/null

echo "[ok] frontend api health proxy"

CHECK_JSON=$(curl -fsS --get "http://127.0.0.1:8002/api/check" \
  --data-urlencode "country=US" \
  --data-urlencode "zip=10013" \
  --data-urlencode "url=https://www.amazon.com/dp/B0DHCZBKW7")

echo "$CHECK_JSON" | grep -q '"country":"US"'
echo "$CHECK_JSON" | grep -q '"free_shipping_country"'
echo "$CHECK_JSON" | grep -q '"method"'

echo "[ok] frontend->backend check path"
echo "Smoke test passed ✅"
