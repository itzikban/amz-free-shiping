# AMZ Free Shipping Checker (Country-Aware)

A Go backend that checks Amazon product pages and answers one specific question reliably:

**"Is this item free-shipping for this destination?"**

This project is designed for automation use-cases (alerts, routines, tracking) where destination matters.

---

## What this app does

- Accepts an Amazon product URL (or ASIN)
- Checks shipping signals with destination context
- Returns structured JSON:
  - `free_shipping`
  - `free_shipping_country`
  - `signal`
  - `method`
- Uses **Decodo scraper API** as primary source (MVP integration)
- Falls back to local browser/HTTP heuristics when needed

### Why this matters
Generic "free shipping" text is not enough. Shipping can differ by country/ZIP. This service is built to avoid false positives.

---

## Current country behavior (MVP)

- `country=US` + `zip=10013`:
  - can return `free_shipping_country=true` if free-shipping signals exist
- `country=IL`:
  - returns `free_shipping_country=false` unless explicit IL/Israel signal is present

This prevents "true for both countries" errors.

---

## API

### Health
```bash
GET /health
```

### Check
```bash
GET /check?country=US&zip=10013&url=https://www.amazon.com/dp/B0DHCZBKW7
```

Example response:
```json
{
  "url": "https://www.amazon.com/dp/B0DHCZBKW7",
  "country": "US",
  "checked_at": "2026-03-09T23:42:43Z",
  "free_shipping": true,
  "free_shipping_country": true,
  "signal": "us_zip_free_shipping_detected",
  "method": "decodo"
}
```

---

## Setup

### 1) Requirements
- Go installed
- Network access
- Decodo account (trial works)

### 2) Environment
Copy and edit:
```bash
cp .env.example .env
```
Set:
- `DECODO_BASIC_AUTH` = Base64 of `username:password`

> Do **not** commit secrets.

### 3) Run
```bash
go mod tidy
go run ./cmd/server
```

Server default: `:8085`

---

## Tests

Run unit tests:
```bash
go test ./...
```

Manual integration test guide:
- `tests/TESTS.md`
- `tests/test-cases.md`

---

## Project structure

- `cmd/server/` - HTTP server entrypoint
- `internal/httpapi/` - API routes
- `internal/checker/` - shipping detection logic
- `tests/` - manual test plans
- `testdata/` - parser fixture samples

---

## Security notes

- Keep API credentials in environment variables only.
- Rotate credentials if they are ever shared in chat or logs.
- This repo intentionally does not store provider secrets.

---

## Roadmap

- Stronger destination-specific parsers per country
- Confidence scoring
- Snapshot storage + change detection
- Alert pipeline (email/telegram/webhook)
- Queue/scheduler integration for large-scale monitoring
