# ITZ-10 Amazon shipping eligibility checker

Status: Closed ✅

## Scope covered
- Switched primary fetch path to Decodo scraper API for Amazon product retrieval.
- Added captcha/blocked-page detection signals.
- Added destination-aware parsing logic (`US` with ZIP support, strict country signal gating).
- Kept fallback flow (`http` + browser) when Decodo is unavailable.

## Deliverables
- [x] Code changes
- [x] Tests added/updated (`internal/checker` path covered)
- [x] Manual verification notes (documented in backend/docs + runtime checks)

## Verification
- [x] `go test ./...` (backend)
- [x] Endpoint/manual flow tested for country-aware checks

## Notes
- Core implementation is aligned with next missions:
  - ITZ-36 (ASIN/canonical normalization)
  - ITZ-12/34 (notification pipeline on monitor diffs)
- Decodo is used as the page-acquisition layer; shipping decision logic remains in backend parser rules.
