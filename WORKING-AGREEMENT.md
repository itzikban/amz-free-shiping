# Working Agreement & Progress Summary

_Last updated: 2026-03-10 (UTC)_

## 1) What has been completed

### Product / Repo foundation
- Repository is active: `itzikban/amz-free-shiping`
- Monorepo structure in place:
  - `backend/` (Go service)
  - `frontend/` (Next.js app)
  - `docs/features/` (feature implementation notes)

### Backend shipped
- Country-aware shipping check endpoint (`/check`) with strict destination logic.
- Decodo integration for Amazon product data fetch.
- Monitor test flow APIs:
  - start/stop/list monitors
  - monitor notifications
  - max-runs auto-stop support
- Price extraction added and improved with heuristics/tests.
- User panel v1 APIs (local user `test-user`):
  - `GET /v1/me`
  - `GET/POST /v1/me/tracked-items`
  - `GET /v1/me/alerts`

### Frontend shipped
- Responsive checker UI with:
  - URL input
  - country + zip
  - strict result display
  - monitor controls (interval, max runs)
  - monitor history + UI notifications
- User panel v1 UI:
  - local user display
  - tracked products list
  - alerts feed

### Process and quality
- Linear tickets created and expanded into sub-issues.
- CodeRabbit configured and used in review loops.
- Added smoke test script (`tests/smoke-ui-backend.sh`) for UI↔backend sanity checks.

### Current active backend feature
- ITZ-19 (Admin metrics/actions) started on branch:
  - `feat/ITZ-19-admin-metrics`
  - PR: https://github.com/itzikban/amz-free-shiping/pull/10

---

## 2) How we want to work together (operating rules)

### Delivery style
1. Work in **small visible chunks**.
2. Every progress update must include:
   - commit hash
   - changed files (or summary)
   - PR link (if opened)
3. No “progress” messages without real git evidence.

### Branch / PR policy
- One feature branch per ticket (or clear scope chunk).
- Open PR early, keep iterating in same PR.
- Run CodeRabbit review loop on each PR.

### CodeRabbit loop policy
- Trigger review after meaningful commits.
- Read all bot feedback (summary + inline comments).
- Fix actionable comments directly, push, retrigger.
- When clean: summarize
  - resolved comments
  - anything intentionally deferred
  - ask explicitly: **"Do you want me to merge?"**

### Testing policy
Before claiming “ready”:
- Backend tests: `go test ./...`
- Frontend build: `npm run build`
- Smoke path (when relevant): `tests/smoke-ui-backend.sh`
- Verify running URLs before sending links.

### Communication policy
- Keep status concise and factual.
- Avoid noisy heartbeat-style updates.
- If blocked, state blocker + exact next action.

---

## 3) What to do next (recommended)

### Immediate
- Continue ITZ-19 PR #10:
  - replace admin action stubs with real logic
  - add minimal admin UI surface for metrics
  - run tests + CodeRabbit cycle

### Then
- ITZ-20 (Admin FE dashboard)
- ITZ-22/23 enhancements (customer management depth)
- ITZ-25+ (price snapshots + subscriptions + alert triggers)

---

## 4) Definition of done for each PR

A PR is "done" only when all are true:
1. Feature scope implemented as agreed.
2. Tests/build pass.
3. CodeRabbit actionable comments resolved or explicitly justified.
4. Final summary sent.
5. Merge approved by user.
