# ITZ-41 — Watchlist + Product Details UX (Frontend)

## Scope implemented
- Added dedicated user-facing **Watchlist page** at `/products`.
- Added dedicated **Product details page** at `/products/[id]`.
- Introduced top navigation links in the app shell to access checker/watchlist/alerts.
- Kept API contract aligned with backend by consuming existing routes:
  - `GET /api/v1/me/tracked-items`
- No backend contract changes were required.

## UX behavior
- Watchlist page shows:
  - Total tracked products
  - Free-shipping count
  - Not-free count
  - Per-item row with destination verdict and last check timestamp
  - Link to per-item details view
- Product detail page shows:
  - ID, URL, country, ZIP, last price, free-shipping verdict, signal/method, and last checked timestamp

## Fallback behavior
When backend is unavailable, UI falls back to local mock data (`frontend/lib/watchlist.ts`) so design/dev reviews can continue without API availability.

## Files touched
- `frontend/app/layout.tsx`
- `frontend/app/products/page.tsx`
- `frontend/app/products/[id]/page.tsx`
- `frontend/app/globals.css`
- `frontend/lib/watchlist.ts`
