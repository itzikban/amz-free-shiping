# ITZ-42 — Alerts Center UX (Frontend)

## Scope implemented
- Added dedicated **Alerts Center** page at `/alerts`.
- Implemented in-app alerts listing UX with client-side filters:
  - All alerts
  - Free shipping alerts
  - Price-change alerts
- Kept API contract aligned with backend by consuming existing routes:
  - `GET /api/v1/me/alerts`

## UX behavior
- Alerts Center includes:
  - Header and context copy
  - Filter buttons for quick signal triage
  - Timestamped alert feed cards
  - Empty-state message when no alerts match filter

## Fallback behavior
When backend is unavailable, UI falls back to local mock alerts (`frontend/lib/watchlist.ts`) to keep the page functional in dev/demo environments.

## Files touched
- `frontend/app/alerts/page.tsx`
- `frontend/lib/watchlist.ts`
- `frontend/app/layout.tsx`
- `frontend/app/globals.css`
