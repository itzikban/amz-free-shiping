# Product Planning Request: AMZ Free Shipping Checker

I want to plan and implement a full product called **AMZ Free Shipping Checker**.

## Product goal

Build a user-friendly web app where a user pastes an Amazon product URL and selects a destination country. The system checks whether the product currently has free shipping to that country.

If the product does not currently have free shipping, the system should continue monitoring it and notify the user when:
- free shipping becomes available again
- the price changes
- the product becomes unavailable or available again

The first version should be practical and MVP-focused, but the long-term direction should support:
- AI-based similar product recommendations
- alternatives that already have free shipping to the user’s country
- better buying insights

## Current status

There is already an early working MVP/test console backed by a Go backend.

Existing capabilities already visible in the current UI:
- backend online / health indication
- product URL input
- destination country selector
- ZIP input for US-only cases
- monitor interval (seconds)
- max runs / auto-stop
- check free shipping action
- start monitor action
- add product to test-user panel action
- local test user panel
- local alerts area
- monitoring test area
- UI notifications area

This means the product already has the beginning of:
- shipping check workflow
- monitor workflow
- local user tracking concept
- notifications concept

## What I need from this planning task

Please plan this product as a **real production-ready application**, while respecting the current MVP direction and not overengineering the first phase.

The plan should cover:

1. Product requirements document
2. Feature breakdown
3. User roles and permissions
4. User stories
5. Admin stories
6. Functional requirements
7. Non-functional requirements
8. System architecture
9. Frontend architecture in Next.js
10. Backend architecture in Go
11. Database schema
12. API design
13. Background jobs / monitor scheduler design
14. Notification design
15. Google login/auth flow
16. Admin panel design
17. User panel design
18. Edge cases
19. Risks / technical unknowns
20. Delivery roadmap
21. Suggested repo structure
22. Suggested phased implementation plan
23. Suggested acceptance criteria for MVP

---

# Product vision

## Core value proposition

Users want to know:
- Does this Amazon item ship free to my country?
- If not now, when will it?
- Did the price change while I was waiting?
- Are there similar products that are cheaper or ship free?

## Product positioning

This product starts as:
- a country-aware shipping checker
- a monitoring and alerting product
- a price tracking companion

Later it becomes:
- a smart shopping assistant with recommendation features

---

# Scope

## MVP scope

### User features
- Sign in with Google
- Paste Amazon product URL
- Select destination country
- Enter ZIP/postal code when required
- Check current shipping status
- Check current price
- Check availability
- Save product to watchlist
- Start monitoring the product
- Set alert preferences
- View tracked products
- View alerts
- Remove a tracked product
- Pause/resume monitoring

### Admin features
- Admin login / role-based access
- View all users
- View all tracked products
- View active monitors
- View failed jobs
- View recent alerts sent
- Manage supported countries
- Manage monitoring defaults
- Manage notification templates
- See basic analytics
- Enable/disable experimental AI recommendation features

## Post-MVP scope
- Price history graph
- Shipping history graph
- Multiple marketplaces
- Better monitoring policies
- Email notifications
- Push notifications
- AI recommendations
- Similar products with free shipping
- Personalized suggestions

---

# Roles

## User
Can:
- sign in with Google
- manage own tracked products
- manage own alerts and settings
- view own notifications

## Admin
Can:
- do everything user can
- manage system-wide settings
- inspect monitors, jobs, users, analytics
- manage supported countries and rules
- manage notification templates
- manage feature flags

---

# Main user flows

## Flow 1: Check product once
1. User signs in
2. User pastes Amazon URL
3. User selects destination country
4. User adds ZIP/postal code if needed
5. System returns:
   - title
   - image
   - current price
   - shipping status
   - shipping cost if known
   - availability
   - last checked time

## Flow 2: Save product and monitor it
1. User performs a check
2. User clicks “Track product”
3. User chooses monitoring preferences
4. Product is saved in watchlist
5. Background monitor checks it periodically
6. System stores snapshots and triggers notifications on changes

## Flow 3: Alert user
1. Monitor detects a change
2. System compares current state to previous state
3. If relevant change occurred:
   - free shipping became available
   - price changed
   - availability changed
4. Notification is created
5. User sees in-app notification
6. Optional email sent in later phase

## Flow 4: Admin operations
1. Admin logs in
2. Admin views dashboards:
   - users
   - monitors
   - failures
   - alerts
3. Admin changes system settings
4. Admin reviews feature flags and countries

---

# Functional requirements

## Authentication
- Users authenticate with Google OAuth
- Session-based or token-based auth supported
- Roles supported: user, admin
- New users created on first login
- Admin role managed internally

## Product check
- Accept Amazon product URL
- Validate URL format
- Parse product identifier
- Fetch/check product metadata
- Determine shipping eligibility for selected country
- Store result snapshot

## Monitoring
- Each tracked product has a monitoring configuration
- Monitors run on interval
- Monitors can auto-stop after max runs if configured
- Monitors can be paused/resumed
- Failures go to retry logic
- Monitor history is auditable

## Notifications
- In-app notifications in MVP
- Notification types:
  - free_shipping_available
  - price_changed
  - availability_changed
  - monitor_failed
- Notifications stored in DB
- Read/unread state supported

## User panel
- Dashboard
- Tracked products list
- Product details page
- Alerts page
- Settings page

## Admin panel
- User management
- Tracked product overview
- Monitoring jobs
- Failures / retries
- Countries / rules
- Feature flags
- Basic analytics

---

# Non-functional requirements

- Clear separation between frontend, backend, background jobs
- Secure authentication
- Role-based access control
- Auditable monitoring state
- Reasonable retry behavior
- Observability: logs, metrics, job status
- API versioning
- Environment-based configuration
- Production-ready error handling
- Scalable enough for moderate user growth

---

# Recommended tech stack

## Frontend
- Next.js (App Router)
- TypeScript
- Tailwind CSS
- shadcn/ui or equivalent component system
- React Query for server state
- NextAuth or Google OAuth integration compatible with backend auth approach

## Backend
- Go
- HTTP API using Gin / Fiber / Echo / chi
- PostgreSQL
- Redis for queues/cache if needed
- Background worker for monitoring jobs
- Structured logging
- Config via env

## Infra
- Docker
- Docker Compose for local dev
- Cloud deployment ready
- Cron/job worker or queue worker
- Optional object storage for assets later

---

# Recommended architecture

## High level services
1. Next.js frontend
2. Go API backend
3. Monitoring worker
4. PostgreSQL database
5. Redis optional for jobs/queue/cache
6. Notification service abstraction

## Architecture principles
- API-first
- stateless web nodes
- workers handle monitoring
- DB stores source of truth
- background jobs produce snapshots + notifications
- feature-flag future AI modules

---

# Database design

Core tables/entities:
- users
- sessions
- roles
- products
- tracked_products
- countries
- shipping_snapshots
- price_snapshots
- monitoring_jobs
- monitoring_runs
- notifications
- notification_preferences
- system_settings
- feature_flags

## Important notes
- products should be normalized by product identifier/ASIN if possible
- tracked_products belongs to user + product + country
- snapshots should be append-only history
- notifications reference the event that triggered them
- monitoring_runs should store success/failure and timing

---

# API expectations

Please design REST APIs for:
- auth
- current user
- product checks
- tracked products
- monitoring actions
- notifications
- admin operations
- settings

Include:
- request/response examples
- status codes
- auth requirements
- validation rules

---

# Frontend information architecture

## Public/auth
- /login
- /auth/callback

## User panel
- /app/dashboard
- /app/products
- /app/products/[id]
- /app/alerts
- /app/settings

## Admin panel
- /admin
- /admin/users
- /admin/products
- /admin/monitors
- /admin/alerts
- /admin/countries
- /admin/settings
- /admin/feature-flags

---

# UI direction

The current product is visually closer to a technical MVP console. The next version should evolve from that, not replace it completely.

## Design direction
- modern dark dashboard
- preserve technical clarity
- cards for check/monitor actions
- table/list for tracked products
- alert feed / notification center
- clear result card after each check
- status badges:
  - free shipping
  - paid shipping
  - unavailable
  - checking
  - monitor active
  - monitor paused
  - failed

## Key screens
1. Check product screen
2. Tracked products screen
3. Product details screen
4. Alerts center
5. Admin monitor dashboard
6. Admin settings/countries screen

---

# Future AI phase

Plan for future modules but do not force them into MVP:
- similar product recommendation engine
- alternative products already shipping free
- ranking by similarity + price + delivery benefit
- buy now vs wait suggestions
- personalized recommendation feed

These should be behind feature flags.

---

# Important planning constraint

Please keep the MVP aligned with the current working functionality already present in the Go backend and current frontend test console.

Do not jump directly into a huge enterprise dashboard. Instead:
- define current MVP foundation
- define production MVP
- define V2
- define AI phase

---

# Deliverables I want from you

Please produce:
1. Full PRD
2. Full system design
3. Suggested DB schema
4. Suggested API contract
5. Frontend module breakdown
6. Backend module breakdown
7. Background worker design
8. Google login integration plan
9. Admin panel plan
10. User panel plan
11. Roadmap by phase
12. Implementation task breakdown by milestone
13. Acceptance criteria per milestone
14. Risks and assumptions
15. Suggested repo/folder structure for Next.js + Go

Please write it in a way that engineering can immediately turn it into implementation tickets.