# ITZ-23 User management UI (v1 local user)

## Scope implemented
- Local seeded user abstraction (`test-user`) via user panel backend service.
- First user flow: add product to track.
- User panel shows tracked products and alert feed.
- Keep API abstraction ready for future UI changes.

## Backend
- Added `internal/userpanel/service.go` (in-memory service abstraction):
  - `Me()`
  - `AddTrackedItem()`
  - `ListItems()`
  - `ListAlerts()`
- Added endpoints:
  - `GET /v1/me`
  - `GET /v1/me/tracked-items`
  - `POST /v1/me/tracked-items`
  - `GET /v1/me/alerts`

## Frontend
- Added route proxies:
  - `/api/v1/me`
  - `/api/v1/me/tracked-items`
  - `/api/v1/me/alerts`
- Added user panel section on homepage:
  - current user display
  - tracked products list
  - user alerts list
  - action button: `Add product to test-user panel`

## Verification
- `go test ./...` ✅
- `npm run build` ✅
- Manual check:
  - add tracked item -> appears in user tracked list
  - alert appears in user alert feed
