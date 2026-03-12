#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="/home/ubuntu/.openclaw/workspace/amz-free-shiping"
CRON_ID="35c0d9c5-a6be-45b0-af30-10c6c2cc313a"
STATE_FILE="$REPO_DIR/.autofix-cron-state"
LOG_FILE="$REPO_DIR/.autofix-cron-toggle.log"

cd "$REPO_DIR"

open_count=$(gh pr list --state open --json number --jq 'length' 2>/dev/null || echo 0)
open_count=${open_count:-0}

current_state="unknown"
if openclaw cron list | grep -q "$CRON_ID"; then
  # parse status column by matching job id line
  status_line=$(openclaw cron list | awk -v id="$CRON_ID" '$1==id {print $0}')
  if echo "$status_line" | grep -q " running "; then
    current_state="enabled"
  elif echo "$status_line" | grep -q " idle "; then
    current_state="enabled"
  elif echo "$status_line" | grep -q " error "; then
    current_state="enabled"
  elif echo "$status_line" | grep -q " disabled "; then
    current_state="disabled"
  fi
fi

prev_intent=""
[ -f "$STATE_FILE" ] && prev_intent=$(cat "$STATE_FILE" || true)

if [ "$open_count" -gt 0 ]; then
  desired="enabled"
  if [ "$current_state" = "disabled" ] || [ "$prev_intent" != "enabled" ]; then
    openclaw cron edit "$CRON_ID" --enable >/dev/null 2>&1 || true
    echo "[$(date -u +'%Y-%m-%dT%H:%M:%SZ')] enabled autofix cron (open_prs=$open_count)" >> "$LOG_FILE"
  fi
else
  desired="disabled"
  if [ "$current_state" != "disabled" ] || [ "$prev_intent" != "disabled" ]; then
    openclaw cron edit "$CRON_ID" --disable >/dev/null 2>&1 || true
    echo "[$(date -u +'%Y-%m-%dT%H:%M:%SZ')] disabled autofix cron (open_prs=0)" >> "$LOG_FILE"
  fi
fi

echo "$desired" > "$STATE_FILE"
