# ITZ-19 Admin metrics + operations API (chunk 1)

Status: Closed ✅

## Implemented
- Added admin metrics service abstraction:
  - `backend/internal/admin/metrics.go`
- Added admin action stubs:
  - `backend/internal/admin/actions.go`
- Added unit test for metrics aggregation:
  - `backend/internal/admin/metrics_test.go`
- Wired monitor/user providers to admin service:
  - `monitor.Service.MonitorCounts()`
  - `userpanel.Service.UserCounts()`
- Exposed admin endpoints:
  - `GET /v1/admin/metrics`
  - `POST /v1/admin/actions/replay-failed-jobs`
  - `POST /v1/admin/actions/retry-failed-notifications`

## Verification
- go test ./...

## Next chunk
- Implement real queue replay/retry logic (replace stubs)
- Add admin UI panel in frontend for these metrics/actions
