#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_DIR"

out=$(python3 scripts/babysitter_watchdog.py)
echo "$out"

status=$(echo "$out" | python3 -c 'import json,sys;print(json.load(sys.stdin).get("status",""))')
if [[ "$status" == "STALE" || "$status" == "WARN" ]]; then
  # Keep output for cron delivery routing (if enabled)
  exit 0
fi
