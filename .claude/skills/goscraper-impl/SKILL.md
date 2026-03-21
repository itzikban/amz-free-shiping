---
name: goscraper-impl
description: "Autonomously implements goscrape (direct Amazon scraper, no PA-API), validates with a live scrape test, builds, tests, and pushes to feat/goscraper-integration. Opus writes and fixes code. Loop until green."
version: "1.0.0"
topology: goscraper-impl
patterns:
  - pipeline
---

# Goscraper-Impl Orchestrator

You are the orchestrator for the `goscraper-impl` topology. Run the full implementation pipeline from start to finish. Do NOT ask the user for permission at any step. Do NOT pause for confirmation. Run everything autonomously.

---

## Step 1 — Create branch

Use the Agent tool to invoke the `branch-creator` agent:

> Follow the instructions in .claude/agents/branch-creator/AGENT.md exactly.
> Create the feat/goscraper-integration branch and confirm success.

---

## Step 2 — Implement goscrape

Use the Agent tool to invoke the `goscraper-implementer` agent:

> Follow the instructions in .claude/agents/goscraper-implementer/AGENT.md exactly.
> Write all four files (goscrape.go, goscrape-test/main.go, edit checker.go, edit fill_to_threshold_service.go).
> Run go build ./... at the end.
> Output status=done or status=error.

**Track `impl_fix_count` starting at 0.**

If status=error → increment `impl_fix_count`. If impl_fix_count >= 1, stop and tell the user:
> "Implementer failed and fixer could not recover. Manual intervention required."
Otherwise go to **Step 3 (fixer for impl error)**.

---

## Step 3 (only if impl error) — Fix implementation error

Use the Agent tool to invoke the `goscraper-fixer` agent, passing the error output:

> Follow the instructions in .claude/agents/goscraper-fixer/AGENT.md exactly.
> Fix the build error: <paste error from Step 2>

Then return to **Step 4**.

---

## Step 4 — Validate scraper live

Use the Agent tool to invoke the `scraper-validator` agent:

> Follow the instructions in .claude/agents/scraper-validator/AGENT.md exactly.
> Run the goscrape-test CLI against Amazon and determine if it works.
> Output status=works, status=captcha_blocked, or status=broken.

**Track `scraper_fix_count` starting at 0.**

- **status=works** → go to **Step 5**.
- **status=captcha_blocked** → go to **Step 5** (CAPTCHA is Amazon's problem, not a code bug — proceed to build/test).
- **status=broken** → increment `scraper_fix_count`. If scraper_fix_count >= 3, stop and tell the user:
  > "Scraper broken after 3 fix attempts. Manual debugging required."
  Otherwise → go to **Step 3b (fixer for broken scraper)**.

---

## Step 3b — Fix broken scraper

Use the Agent tool to invoke the `goscraper-fixer` agent:

> Follow the instructions in .claude/agents/goscraper-fixer/AGENT.md exactly.
> The live scraper test failed. Fix the goscrape.go implementation: <paste error from Step 4>

Then return to **Step 4**.

---

## Step 5 — Build and test

Use the Agent tool to invoke the `builder-tester` agent:

> Follow the instructions in .claude/agents/builder-tester/AGENT.md exactly.
> Run go build ./... and go test ./internal/checker/... -timeout 60s.
> Output status=green or status=red.

**Track `build_fix_count` starting at 0.**

- **status=green** → go to **Step 6**.
- **status=red** → increment `build_fix_count`. If build_fix_count >= 5, stop and tell the user:
  > "Build/tests failing after 5 fix attempts. Manual debugging required."
  Otherwise → go to **Step 5b (fixer for build failure)**.

---

## Step 5b — Fix build/test failure

Use the Agent tool to invoke the `goscraper-fixer` agent:

> Follow the instructions in .claude/agents/goscraper-fixer/AGENT.md exactly.
> Fix the build/test failure: <paste error output from Step 5>

Then return to **Step 5**.

---

## Step 6 — Commit and push

Use the Agent tool to invoke the `goscraper-commit-pusher` agent:

> Follow the instructions in .claude/agents/goscraper-commit-pusher/AGENT.md exactly.
> Stage the goscraper files, commit, and push feat/goscraper-integration.

---

## Step 7 — Done

Tell the user:
> "feat/goscraper-integration is pushed. The goscrape method is ready.
>
> Morning verification:
>   git log feat/goscraper-integration --oneline
>   cd backend && go run ./cmd/goscrape-test/ 'wireless earbuds'
>   ALT_FETCH_METHOD=goscrape go run ./cmd/server/"
