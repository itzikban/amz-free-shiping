# ITZ-34 — [NOTIFY] In-app notification center + read/unread + preferences

## Scope delivered

Implemented MVP in-app notifications flow for local test user (`test-user`) with:

- Notification feed endpoint
- Read/unread behavior (single + mark all)
- User notification preferences endpoint
- Frontend notification center UI with inbox + unread badge + preference toggles

## Backend

### New user APIs

- `GET /v1/me/notifications?unread=true|false`
- `POST /v1/me/notifications/read` with body:
  - `{ "id": "notif-..." }` (single)
  - `{ "all": true }` (bulk)
- `GET /v1/me/notification-preferences`
- `PUT /v1/me/notification-preferences`

### Service behavior

- Added in-memory `Notification` model: `id`, `title`, `message`, `read`, `created_at`, `read_at`
- Added in-memory `NotificationPreferences` model:
  - `in_app_enabled`
  - `on_item_added`
- `AddTrackedItem` now emits in-app notifications when preferences allow.

## Frontend

### Proxy routes

- `frontend/app/api/v1/me/notifications/route.ts`
- `frontend/app/api/v1/me/notifications/read/route.ts`
- `frontend/app/api/v1/me/notification-preferences/route.ts`

### UI additions (`frontend/app/page.tsx`)

- New “In-app notifications center” card
- Inbox list with unread indicator
- Per-notification “mark read” button
- “mark all read” action
- Preferences controls:
  - enable/disable in-app notifications
  - enable/disable item-added event notifications

## Tests

- Added `backend/internal/userpanel/service_notifications_test.go`
  - validates notification creation
  - validates read transition
  - validates preference disable behavior

## Notes

- This is MVP behavior using existing in-memory userpanel service (consistent with current project architecture).
- No DB migration required for this ticket’s current scope.
