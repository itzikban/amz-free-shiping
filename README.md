# amz-free-shiping (Monorepo)

![CodeRabbit Pull Request Reviews](https://img.shields.io/coderabbit/prs/github/itzikban/amz-free-shiping?utm_source=oss&utm_medium=github&utm_campaign=itzikban%2Famz-free-shiping&labelColor=171717&color=FF570A&link=https%3A%2F%2Fcoderabbit.ai&label=CodeRabbit+Reviews)

This repository is structured as a **monorepo** for multiple services.

## Current layout

- `backend/` — Go backend service (free-shipping checker API)
- `frontend/` — Next.js responsive web app

## Planned expansion

- `backend/` — API + workers + schedulers
- `frontend/` — dashboard + user flows
- additional services as needed (`services/*`)

## Backend quick start

```bash
cd backend
cp .env.example .env
# set DECODO_BASIC_AUTH in .env or environment

go mod tidy
go run ./cmd/server
```

API runs on `:8085` by default.

## Implemented features

### Backend
- Country-aware `/check` endpoint (`US` with ZIP, `IL` strict destination logic)
- Decodo integration as primary scraper source
- Captcha/blocked-page detection signaling
- Initial PostgreSQL schema migration (`backend/migrations/001_init.sql`)
- Redis queue runtime with retry + dead-letter flow
- Scheduler service for enqueueing due checks

### Frontend
- Responsive Next.js app in `frontend/`
- URL + country + ZIP check form
- Backend health indicator (live polling)
- Strict result display (`free_shipping` vs `free_shipping_country`)
- Raw backend response viewer for debugging

## Notes
- Feature implementation docs are tracked under `docs/features/`.
- End-to-end smoke check script: `tests/smoke-ui-backend.sh`


## Working process
- See `WORKING-AGREEMENT.md` for collaboration rules, delivery workflow, and definition of done.
