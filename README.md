# amz-free-shiping (Monorepo)

![CodeRabbit Pull Request Reviews](https://img.shields.io/coderabbit/prs/github/itzikban/amz-free-shiping?utm_source=oss&utm_medium=github&utm_campaign=itzikban%2Famz-free-shiping&labelColor=171717&color=FF570A&link=https%3A%2F%2Fcoderabbit.ai&label=CodeRabbit+Reviews)

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

## Dev note

This line is added to validate PR automation (CodeRabbit + Linear).
