# Test Guide

This folder contains test guidance for validating country-specific free-shipping detection.

## Goal
The checker must return different results per destination context:
- US ZIP 10013 -> can be `true`
- IL (Israel) -> should be `false` unless explicit Israel free-shipping evidence exists

## Local run
```bash
go test ./...
DECODO_BASIC_AUTH=your_base64 go run ./cmd/server
```

## API checks

### 1) US check (ZIP context)
```bash
curl --get "http://127.0.0.1:8085/check" \
  --data-urlencode "country=US" \
  --data-urlencode "zip=10013" \
  --data-urlencode "url=https://www.amazon.com/dp/B0DHCZBKW7"
```
Expected (for current known sample):
- `free_shipping_country: true`

### 2) IL check
```bash
curl --get "http://127.0.0.1:8085/check" \
  --data-urlencode "country=IL" \
  --data-urlencode "url=https://www.amazon.com/dp/B0DHCZBKW7"
```
Expected (for current known sample):
- `free_shipping_country: false`

## Validation rules
- Do not alert users from generic "free shipping" text only.
- Alert only when destination-specific logic is satisfied.
- For IL, require explicit Israel evidence before alerting.

## Notes
- Amazon responses vary by session, anti-bot filtering, and geography.
- Decodo scraper API is used as primary source in this MVP.
