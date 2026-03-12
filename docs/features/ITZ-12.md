# ITZ-12 Outbox-based alert delivery

Status: In Progress

## Scope implemented
- Added outbox notification service with:
  - idempotency key dedup
  - pending/delivered/failed states
  - retry scheduling with backoff (`next_attempt_at`)
- Integrated userpanel alerts with outbox enqueue + dispatch.
- Added transition alert behavior for tracked items:
  - emits `free_shipping_available` when state changes `NOT_FREE -> FREE`.
- Implemented `RetryFailedNotifications` to process due outbox entries and sync delivered in-app notifications.

## Verification
- `go test ./internal/notify ./internal/userpanel`

## Notes
- This ticket focuses on reliable in-app delivery path and transition-based alerts.
- External channels (email/push) can reuse the same outbox model later.
