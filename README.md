# amz-free-shiping (Monorepo)

This repository is structured as a **monorepo** for multiple services.

## Current layout

- `backend/` — Go backend service (free-shipping checker API)

## Planned layout

- `backend/` — API + workers + schedulers
- `frontend/` — web app/dashboard
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
