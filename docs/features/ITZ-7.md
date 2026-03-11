# ITZ-7 Create PostgreSQL schema + migrations

Status: Closed ✅

## Scope
Implement the initial relational schema required for tracking, snapshots, alerts, and outbox delivery.

## Deliverables
- [x] Initial migration file at `backend/migrations/001_init.sql`
- [x] Core tables: users, tracked_items, snapshots, alternatives, alerts, outbox, notification_attempts
- [x] Useful indexes for due checks and alert retrieval
- [x] DB connection helper (`backend/internal/store/store.go`)
- [x] Schema presence test (`backend/internal/store/schema_test.go`)

## Verification
- [x] `go test ./...` from `backend/`
- [x] Migration includes all required tables and indexes

## Notes
- Branch: `feat/ITZ-7`
- DSN is read from `DATABASE_URL` env var
- Future step: add migration runner CLI and rollback migrations
