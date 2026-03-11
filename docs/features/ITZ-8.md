# ITZ-8 Setup Redis job queues and worker runtime

Status: Closed ✅

## Scope
Introduce a queue runtime for check/notify jobs with retry and dead-letter behavior.

## Deliverables
- [x] Queue job types and payload contracts (`backend/internal/queue/types.go`)
- [x] Redis queue implementation (`backend/internal/queue/redis_queue.go`)
- [x] Worker runtime with handler dispatch + retry/dead-letter (`backend/internal/queue/worker.go`)
- [x] Unit tests for payload and worker guardrails (`backend/internal/queue/queue_test.go`)

## Verification
- [x] `go test ./...` from `backend/`

## Notes
- Branch: `feat/ITZ-8`
- Uses Redis lists for main queue, sorted set for delayed retries, and dedicated DLQ list.
- Future step: add metrics and dedicated retry poller process.
