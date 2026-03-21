---
name: perf-ux-improvements
description: "Parallel performance and UX fix agents for the amz-free-shipping app — engineers consult each other when needed, QA routes fixes back to the right engineer."
version: "2.0.0"
topology: perf-ux-improvements
patterns:
  - fan-out
---

# Perf Ux Improvements Topology Skill

Parallel performance and UX fix agents for the amz-free-shipping app — engineers consult each other when needed, QA routes fixes back to the right engineer.

Version: 2.0.0
Patterns: fan-out

Domain: engineering
Timeout: 30m

## Flow

- coordinator -> frontend-engineer
- coordinator -> backend-engineer
- frontend-engineer -> qa-reviewer [when frontend-engineer.status == done]
- frontend-engineer -> backend-engineer [when frontend-engineer.status == needs-backend-consultation] [max 1]
- backend-engineer -> qa-reviewer [when backend-engineer.status == done]
- backend-engineer -> frontend-engineer [when backend-engineer.status == needs-frontend-consultation] [max 1]
- qa-reviewer -> reporter [when qa-reviewer.verdict == approved]
- qa-reviewer -> frontend-engineer [when qa-reviewer.verdict == needs-frontend-fix] [max 2]
- qa-reviewer -> backend-engineer [when qa-reviewer.verdict == needs-backend-fix] [max 2]
- qa-reviewer -> frontend-engineer [when qa-reviewer.verdict == needs-both-fix] [max 2]
- qa-reviewer -> backend-engineer [when qa-reviewer.verdict == needs-both-fix] [max 2]

