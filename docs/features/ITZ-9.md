# ITZ-9 Enqueue due tracking checks (scheduler)

Status: Closed ✅

## Scope
Implement scheduler logic that finds due items and enqueues check jobs while updating next-check atomically (interface-level in MVP).

## Deliverables
- [x] Scheduler service (`backend/internal/scheduler/scheduler.go`)
- [x] Pluggable interfaces for due item source + queue enqueuer
- [x] Tick flow:
  - read due items
  - enqueue checks
  - mark next check timestamp
- [x] Unit tests (`backend/internal/scheduler/scheduler_test.go`)

## Verification
- [x] `go test ./...` from backend

## Notes
- Branch: `feat/ITZ-9`
- Next step: bind scheduler interfaces to real Postgres repo + Redis queue implementations.
