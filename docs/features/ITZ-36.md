# ITZ-36 ASIN normalization + canonical product dedup model

Status: In Review ✅

## Scope covered
- Added ASIN normalization for tracked product input (raw ASIN, `/dp/`, `/gp/product/`, `asin=` query param).
- Added canonical product identity fields on tracked items (`product_id`, `asin`, `canonical_url`).
- Added in-memory canonical product registry keyed by normalized ASIN (`asin:<ASIN>`) with URL hash fallback.
- Added dedup model for tracked items by scope:
  - `user + canonical product + country + zip`
  - duplicate add updates latest check on existing tracked item instead of creating a duplicate record.

## Deliverables
- [x] Backend implementation (`internal/userpanel`)
- [x] Unit tests for normalization + dedup behavior
- [x] Feature doc update

## Verification
- [x] `go test ./internal/userpanel ./internal/checker ./internal/httpapi`
- [x] `npm run build` (frontend)

## Notes
- Dedup remains destination-aware (`country`, `zip`) because shipping eligibility can differ per destination.
