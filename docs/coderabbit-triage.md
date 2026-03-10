# CodeRabbit Triage (Merged PRs #4, #5, #6)

## PR #4 (DB migration)
- Suggestion: execute migration test instead of substring-only checks.
  - Decision: **Accepted (partial)**
  - Implemented: SQL parse test added via `pg_query` parser.
  - Follow-up: add full ephemeral Postgres migration execution test.
- Suggestion: add missing FK indexes (`alternatives.tracked_item_id`, `outbox.alert_id`).
  - Decision: **Accepted**
  - Implemented in `backend/migrations/001_init.sql`.

## PR #5 (Queue)
- Suggestion: fix `MaxRetries` off-by-one.
  - Decision: **Accepted**
  - Implemented: DLQ only when `Attempts > MaxRetries`.
- Suggestion: don't ignore worker queue operation errors.
  - Decision: **Accepted**
  - Implemented: explicit logging + handling for drain/pop/retry failures.

## PR #6 (Scheduler)
- Suggestion: avoid enqueue-then-mark race.
  - Decision: **Accepted**
  - Implemented: claim-first flow with `ClaimDueItems`; release claim on enqueue failure.
  - Tests added for failure path.

## Notes
- Batch implementation branch: `feat/ITZ-28-coderabbit-batch`
- A single PR is used to apply all accepted suggestions.
