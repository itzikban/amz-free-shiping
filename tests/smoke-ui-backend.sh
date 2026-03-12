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

export ADMIN_API_TOKEN="${ADMIN_API_TOKEN:-smoke-admin-token}"
export LOCAL_ADMIN_USERNAME="${LOCAL_ADMIN_USERNAME:-admin}"
export LOCAL_ADMIN_PASSWORD="${LOCAL_ADMIN_PASSWORD:-Admin@12345}"

# free ports if occupied by stale processes
fuser -k 8085/tcp 2>/dev/null || true
fuser -k 8002/tcp 2>/dev/null || true

(
  cd "$BACKEND_DIR"
  ADMIN_API_TOKEN="$ADMIN_API_TOKEN" go run ./cmd/server >/tmp/amz-backend.log 2>&1
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
  BACKEND_BASE_URL="http://127.0.0.1:8085" \
  ADMIN_API_TOKEN="$ADMIN_API_TOKEN" \
  LOCAL_ADMIN_USERNAME="$LOCAL_ADMIN_USERNAME" \
  LOCAL_ADMIN_PASSWORD="$LOCAL_ADMIN_PASSWORD" \
  npm run dev -- -p 8002 >/tmp/amz-frontend.log 2>&1
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

COOKIE_JAR="/tmp/amz-admin-cookies.txt"
rm -f "$COOKIE_JAR"
LOGIN_STATUS=$(curl -sS -o /tmp/amz-admin-login.json -w "%{http_code}" \
  -c "$COOKIE_JAR" \
  -H 'content-type: application/json' \
  -X POST "http://127.0.0.1:8002/api/v1/admin/login" \
  --data "{\"username\":\"$LOCAL_ADMIN_USERNAME\",\"password\":\"$LOCAL_ADMIN_PASSWORD\"}")
if [[ "$LOGIN_STATUS" != "200" ]]; then
  echo "[fail] admin login failed with status $LOGIN_STATUS"
  cat /tmp/amz-admin-login.json || true
  exit 1
fi

METRICS_STATUS=$(curl -sS -o /tmp/amz-admin-metrics.json -w "%{http_code}" \
  -b "$COOKIE_JAR" \
  "http://127.0.0.1:8002/api/v1/admin/metrics")
if [[ "$METRICS_STATUS" != "200" ]]; then
  echo "[fail] admin metrics fetch failed with status $METRICS_STATUS"
  cat /tmp/amz-admin-metrics.json || true
  exit 1
fi

grep -q 'monitors_total' /tmp/amz-admin-metrics.json

echo "[ok] admin login + authenticated metrics path"
echo "Smoke test passed ✅"
