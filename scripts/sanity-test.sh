#!/usr/bin/env bash
# sanity-test.sh — Integration sanity tests for the fill-to-threshold feature.
# Usage: bash scripts/sanity-test.sh
# Requires: curl, python3

set -e

BACKEND="http://127.0.0.1:8085"
FRONTEND="http://127.0.0.1:3000"
PASS=0
FAIL=0

pass() { echo "PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "FAIL: $1"; FAIL=$((FAIL + 1)); }

# ---------------------------------------------------------------------------
# 0. Pre-flight: check that services are reachable at all
# ---------------------------------------------------------------------------
if ! curl -sf --max-time 3 "$BACKEND/health" >/dev/null 2>&1; then
  echo "ERROR: backend is not running at $BACKEND — start services first."
  exit 1
fi

if ! curl -sf --max-time 3 "$FRONTEND" >/dev/null 2>&1; then
  echo "ERROR: frontend is not running at $FRONTEND — start services first."
  exit 1
fi

# ---------------------------------------------------------------------------
# 1. Backend health
# ---------------------------------------------------------------------------
echo "--- [1] Backend health ---"
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BACKEND/health")
if [ "$HTTP_STATUS" = "200" ]; then
  pass "backend /health returned 200"
else
  fail "backend /health returned $HTTP_STATUS (expected 200)"
fi

# ---------------------------------------------------------------------------
# 2. Frontend health
# ---------------------------------------------------------------------------
echo "--- [2] Frontend health ---"
FRONT_HEALTH=$(curl -sf --max-time 5 "$FRONTEND/api/health" 2>/dev/null || true)
if echo "$FRONT_HEALTH" | grep -q "ok"; then
  pass "frontend /api/health contains 'ok'"
else
  fail "frontend /api/health response did not contain 'ok': $FRONT_HEALTH"
fi

# ---------------------------------------------------------------------------
# 3. Fill-to-threshold API response structure
# ---------------------------------------------------------------------------
echo "--- [3] Fill-to-threshold API response structure ---"
RESP=$(curl -sf --max-time 15 \
  "$BACKEND/api/v1/fill-to-threshold?url=https://www.amazon.com/dp/B08N5LNQCX&country=US&threshold=50" \
  2>/dev/null || true)

if [ -z "$RESP" ]; then
  fail "fill-to-threshold API returned empty response"
else
  CHECK=$(echo "$RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
except Exception as e:
    print('invalid-json')
    sys.exit(0)
if 'current_price' in d or 'error' in d:
    print('ok')
else:
    print('missing-fields')
" 2>/dev/null || echo "python3-error")

  case "$CHECK" in
    ok) pass "fill-to-threshold API returned valid JSON with 'current_price' or 'error' field" ;;
    invalid-json) fail "fill-to-threshold API response is not valid JSON: $RESP" ;;
    missing-fields) fail "fill-to-threshold response has neither 'current_price' nor 'error': $RESP" ;;
    *) fail "unexpected check result: $CHECK" ;;
  esac
fi

# ---------------------------------------------------------------------------
# 4. Suggestion card URLs are non-empty when suggestions exist
# ---------------------------------------------------------------------------
echo "--- [4] Suggestion URL fields ---"
if [ -z "$RESP" ]; then
  fail "skipped: no API response available"
else
  URL_CHECK=$(echo "$RESP" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
except Exception:
    print('invalid-json')
    sys.exit(0)
sugs = d.get('suggestions', [])
if sugs:
    bad = [s for s in sugs if not s.get('url')]
    if bad:
        print('bad:' + str(len(bad)))
        sys.exit(0)
    print('ok:' + str(len(sugs)))
else:
    print('none')
" 2>/dev/null || echo "python3-error")

  case "$URL_CHECK" in
    ok:*) pass "all ${URL_CHECK#ok:} suggestions have non-empty url fields" ;;
    none) pass "no suggestions returned (normal for this product)" ;;
    bad:*) fail "${URL_CHECK#bad:} suggestion(s) are missing url fields" ;;
    invalid-json) fail "response is not valid JSON" ;;
    *) fail "unexpected url-check result: $URL_CHECK" ;;
  esac
fi

# ---------------------------------------------------------------------------
# 5. Frontend page loads and contains expected content
# ---------------------------------------------------------------------------
echo "--- [5] Frontend page content ---"
PAGE=$(curl -sf --max-time 10 "$FRONTEND" 2>/dev/null || true)
if echo "$PAGE" | grep -qi "amazon"; then
  pass "frontend page contains 'Amazon'"
else
  fail "frontend page did not contain 'Amazon' (got ${#PAGE} bytes)"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
