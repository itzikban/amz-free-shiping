---
name: coderabbit-pingpong
description: "Fully autonomous ping-pong loop across ALL open PRs: scan every PR for CodeRabbit comments, fix all of them, push, wait for re-review, repeat until each PR is clean."
version: "1.0.0"
topology: coderabbit-pingpong
patterns:
  - pipeline
---

# CodeRabbit Ping-Pong — Orchestrator

You are the orchestrator for the `coderabbit-pingpong` topology. When invoked, run the full autonomous loop from start to finish across ALL open PRs. Do NOT ask the user for permission at any step. Do NOT pause for confirmation. Run everything autonomously until every PR is clean.

---

## Step 1 — Scan all open PRs

Use the Agent tool to invoke the `pr-scanner` agent:

> Follow the instructions in .claude/agents/pr-scanner/AGENT.md exactly.
> Find all open PRs in this repo that have unresolved @coderabbit comments.
> Output status=has-prs with a PR_LIST, or status=no-prs.

If status=no-prs → tell the user "No open PRs with CodeRabbit comments found." and stop.

---

## Step 2 — Collect all comments

Use the Agent tool to invoke the `pr-reader` agent, passing the PR list:

> Follow the instructions in .claude/agents/pr-reader/AGENT.md exactly.
> Process this list of PR numbers: <PR_LIST from Step 1>.
> Check out each branch and collect every unresolved @coderabbit comment.
> Output status=has-comments with the full structured comment list, or status=clean.

If status=clean → tell the user "All PRs are already clean." and stop.

---

## Step 3 — Fix everything

Use the Agent tool to invoke the `code-fixer` agent, passing all the comments:

> Follow the instructions in .claude/agents/code-fixer/AGENT.md exactly.
> Fix every single comment in the list below. Do not skip any.
> <paste full comment list from Step 2>
> Output status=fixed with a list of all fixes applied, or status=nothing-to-fix.

If status=nothing-to-fix → go to Step 6 (done).

---

## Step 4 — Commit and push

Use the Agent tool to invoke the `commit-pusher` agent:

> Follow the instructions in .claude/agents/commit-pusher/AGENT.md exactly.
> Stage all changes, commit, push the branch, and assign @coderabbit for re-review.

---

## Step 5 — Wait for CodeRabbit (ping-pong loop)

Track `round` starting at 0 and `trigger_count` starting at 0.

Use the Agent tool to invoke the `coderabbit-watcher` agent:

> Follow the instructions in .claude/agents/coderabbit-watcher/AGENT.md exactly.
> Poll for a new @coderabbit review for up to 2 minutes (every 15 seconds).
> If no review appears after 2 minutes, push an empty trigger commit and output verdict=triggered-retry.
> Output verdict=pr-approved, verdict=new-comments, or verdict=triggered-retry.

**Handle the verdict:**

- **verdict=pr-approved** → go to Step 6 (done).

- **verdict=new-comments** → increment `round`. If round >= 10, stop and tell the user:
  > "Reached 10 fix rounds without full approval. Manual review required."
  > List the remaining comments.
  Otherwise, go back to **Step 2** (re-read all comments on all PRs and fix again).

- **verdict=triggered-retry** → increment `trigger_count`. If trigger_count >= 5, stop and tell the user:
  > "CodeRabbit did not respond after 5 trigger attempts. Check that @coderabbit is installed on this repo."
  Otherwise, go back to the top of **Step 5** (wait again for the review).

---

## Step 6 — Done

Tell the user:
> "All PRs are clean. CodeRabbit approved after <round> fix round(s)."
