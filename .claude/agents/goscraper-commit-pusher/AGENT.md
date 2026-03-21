---
name: goscraper-commit-pusher
description: "Stages the new goscraper files, commits with a descriptive message, and pushes the feat/goscraper-integration branch."
model: haiku
tools:
  - Bash
---

## Instructions

You are a git commit agent. Stage specific files, commit, and push.

Step 1 — Stage only the relevant files:
  git -C /home/ubuntu/.openclaw/workspace/amz-free-shiping add backend/internal/checker/goscrape.go
  git -C /home/ubuntu/.openclaw/workspace/amz-free-shiping add backend/cmd/goscrape-test/
  git -C /home/ubuntu/.openclaw/workspace/amz-free-shiping add backend/internal/checker/checker.go
  git -C /home/ubuntu/.openclaw/workspace/amz-free-shiping add backend/internal/checker/fill_to_threshold_service.go

Step 2 — Check status:
  git -C /home/ubuntu/.openclaw/workspace/amz-free-shiping status --porcelain

Step 3 — Commit:
  git -C /home/ubuntu/.openclaw/workspace/amz-free-shiping commit -m "feat: add goscrape method for Amazon alternatives (no PA-API required)

Direct Amazon scraper using goquery CSS selectors. Adds goscrape as:
- ALT_FETCH_METHOD=goscrape case in enrichWithAlternatives
- Second fallback in BuildFillToThresholdForURL (between Decodo and scrapeAmazonAlternatives)
- cmd/goscrape-test/ CLI for live smoke testing

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"

Step 4 — Push:
  git -C /home/ubuntu/.openclaw/workspace/amz-free-shiping push -u origin feat/goscraper-integration

Print "Pushed feat/goscraper-integration — ready for review." and stop.

