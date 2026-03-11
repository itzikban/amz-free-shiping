# AMZ Free Shipping Checker — Product Planning (Reference)

Last updated: 2026-03-11
Owner: @itzikban

This document captures the planning direction shared in `amz-plan` and maps it to implementation tickets in Linear so future sessions can resume quickly.

## 1) Product goal

Build a country-aware Amazon shipping checker where users can:
- check free shipping for selected country (and ZIP/postal when needed)
- track products over time
- get alerts when shipping/price/availability changes

Long-term direction:
- alternatives and recommendations
- smarter buying insights

## 2) MVP boundaries

### In MVP
- Google login + session
- role model (user/admin)
- product check flow
- watchlist + monitor lifecycle
- in-app notifications
- admin operations panel
- country rules management
- observability baseline

### Post-MVP
- charts/history UX polish
- email/push delivery channels
- AI recommendations and alternatives

## 3) High-level architecture

- **Frontend**: Next.js App Router
- **Backend API**: Go (REST)
- **Worker**: monitor scheduler + queue processing
- **DB**: PostgreSQL (source of truth)
- **Queue/cache**: Redis (jobs/retries/dead-letter)

Core flow:
1. User checks product
2. Product normalized (ASIN/canonical)
3. Track + monitor jobs run periodically
4. Snapshots saved
5. Diffs emit notifications

## 4) Frontend information architecture

- Public/Auth: `/login`, `/auth/callback`
- User: `/app/dashboard`, `/app/products`, `/app/products/[id]`, `/app/alerts`, `/app/settings`
- Admin: `/admin`, `/admin/users`, `/admin/products`, `/admin/monitors`, `/admin/alerts`, `/admin/countries`, `/admin/settings`, `/admin/feature-flags`

## 5) Implementation ticket map (Linear)

### Foundation/Auth/Architecture
- ITZ-32 — [AUTH] Google OAuth + session foundation
- ITZ-33 — [AUTHZ] RBAC middleware + route guards
- ITZ-38 — [ARCH] PRD + API contract freeze

### Core product
- ITZ-10 — shipping eligibility checker
- ITZ-36 — ASIN normalization + canonical dedup model
- ITZ-17 / ITZ-22 / ITZ-23 — customer/tracking management

### Notifications/monitoring
- ITZ-12 — outbox-based alert delivery
- ITZ-34 — in-app notification center + read/unread + preferences

### Admin + governance
- ITZ-16 / ITZ-20 / ITZ-43 — admin dashboard & operations UI
- ITZ-35 — countries/rules management

### Frontend UX program
- ITZ-39 — FE foundation shell + route groups
- ITZ-40 — product check UX
- ITZ-41 — watchlist + product details UX
- ITZ-42 — alerts center UX
- ITZ-44 — design system pass
- ITZ-45 — EN/HE i18n + RTL support

### Ops/quality
- ITZ-37 — observability baseline
- ITZ-14 / ITZ-15 / ITZ-21 / ITZ-24 / ITZ-27 — hardening + QA

## 6) Suggested execution order

1. ITZ-38 + ITZ-32 + ITZ-33
2. ITZ-39 + ITZ-44 + ITZ-45
3. ITZ-10 + ITZ-36
4. ITZ-12 + ITZ-34 + ITZ-42
5. ITZ-41 + ITZ-17/22 polish
6. ITZ-43 + ITZ-35
7. ITZ-37 + QA/ops tracks
8. AI/recommendation phase (ITZ-11 + future tickets)

## 7) UX mock reference (static)

A static visual mock (no functionality) is included at:
- `mock-ui/index.html`

This mock reflects:
- dark dashboard style
- check result card
- tracked products table
- alerts feed
- admin operations snapshot

## 8) Notes for future sessions

- Use this file + Linear ITZ board as the source of truth for planning continuity.
- Keep MVP practical; avoid overengineering before reliability/auth/monitoring are solid.
- Keep AI features behind feature flags until core reliability and UX are stable.
