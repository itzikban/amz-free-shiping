# Babysitter + Watchdog (Clean Automation Layer)

This layer is intentionally isolated from app logic.

## Goals
- Keep branch work aligned to ticket scope.
- Detect inactivity/drift.
- Emit concise status for cron routing.

## Files
- `scripts/babysitter_watchdog.py` — core checker.
- `scripts/run-babysitter-cron.sh` — cron-friendly runner.
- `.agent/branch-map.json` (local-only, ignored) — optional branch->ticket mapping.
- `.agent/watchdog-state.json` (local-only, ignored) — runtime state.

## Branch → Ticket Linking
Preferred:
- Include ticket in branch name: `feat/ITZ-40-...`

Alternative:
- Put mapping in `.agent/branch-map.json`:
```json
{
  "feat/fill-to-50-basket-booster": "ITZ-40"
}
```

## Script Output
`babysitter_watchdog.py` prints one JSON object:
- `status`: `OK` | `WARN` | `STALE`
- `summary`
- `branch`
- `head`
- `ticket`
- `stalled`

## Suggested Cron
Every 10 minutes, isolated session, optional delivery on WARN/STALE.

Example payload command:
```bash
bash scripts/run-babysitter-cron.sh
```

## Important
No frontend/backend application behavior changes are part of this layer.
