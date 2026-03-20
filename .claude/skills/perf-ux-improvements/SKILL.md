---
name: perf-ux-improvements
description: "Parallel performance and UX fix agents for the amz-free-shipping app — engineers consult each other when needed, QA routes fixes back to the right engineer."
version: "2.0.0"
topology: perf-ux-improvements
patterns:
  - fan-out
---

# Perf UX Improvements — Orchestrator

You are the orchestrator for the `perf-ux-improvements` topology. When this skill is invoked, you run the full multi-agent flow from start to finish. Follow the steps below exactly — do not skip steps, do not ask the user for input between steps unless a step explicitly says to.

---

## Step 1 — Coordinator

Use the Agent tool to invoke the `coordinator` agent:

> Read the following files in /home/ubuntu/.openclaw/workspace/amz-free-shiping:
> - frontend/app/page.tsx
> - backend/internal/checker/fill_to_threshold_service.go
> - backend/internal/checker/decodo.go
>
> Then follow the instructions in .claude/agents/coordinator/AGENT.md exactly.

Wait for it to complete. Report its summary to the user as a bullet list.

---

## Step 2 — Parallel: Frontend + Backend Engineers

Launch BOTH agents at the same time using two parallel Agent tool calls in a single message:

**Agent call A** — frontend-engineer:
> Follow the instructions in .claude/agents/frontend-engineer/AGENT.md exactly.
> Work on /home/ubuntu/.openclaw/workspace/amz-free-shiping.
> At the end, output your status as either:
>   status=done
>   status=needs-backend-consultation: <description of what you need>

**Agent call B** — backend-engineer:
> Follow the instructions in .claude/agents/backend-engineer/AGENT.md exactly.
> Work on /home/ubuntu/.openclaw/workspace/amz-free-shiping.
> At the end, output your status as either:
>   status=done
>   status=needs-frontend-consultation: <description of what you need>

Wait for both to finish. Note the status output of each.

---

## Step 3 — Cross-Consultation (if needed, max 1 round)

**If frontend-engineer output `status=needs-backend-consultation`:**
Run a new Agent call for backend-engineer:
> The frontend-engineer needs your input: <paste their consultation message>.
> Address this, make any necessary changes to the backend files, then output status=done.

**If backend-engineer output `status=needs-frontend-consultation`:**
Run a new Agent call for frontend-engineer:
> The backend-engineer needs your input: <paste their consultation message>.
> Address this, make any necessary changes to the frontend files, then output status=done.

If both need consultation from each other simultaneously, run frontend consultation first, then backend.
Skip this step entirely if both outputs were `status=done`.

---

## Step 4 — QA Reviewer

Use the Agent tool to invoke the `qa-reviewer` agent:

> Follow the instructions in .claude/agents/qa-reviewer/AGENT.md exactly.
> Work on /home/ubuntu/.openclaw/workspace/amz-free-shiping.
> At the end, output your verdict as exactly one of:
>   verdict=approved
>   verdict=needs-frontend-fix: <list of issues>
>   verdict=needs-backend-fix: <list of issues>
>   verdict=needs-both-fix: <list of frontend issues> | <list of backend issues>

Show the user the full QA report. Note the verdict.

---

## Step 5 — Fix Loops (max 2 rounds)

Track a `fix_round` counter starting at 0. Repeat this block while verdict ≠ `approved` and fix_round < 2:

1. Increment fix_round.
2. **If `verdict=needs-frontend-fix`:** Run frontend-engineer agent with the list of issues from QA. Output status=done when fixed.
3. **If `verdict=needs-backend-fix`:** Run backend-engineer agent with the list of issues from QA. Output status=done when fixed.
4. **If `verdict=needs-both-fix`:** Run frontend-engineer AND backend-engineer in parallel (two Agent calls), each with their respective issue list.
5. Re-run Step 4 (QA reviewer) to get a new verdict.

If fix_round reaches 2 and verdict is still not `approved`, stop and tell the user:
> "QA found unresolved issues after 2 fix rounds. Manual review required."
> List the remaining issues clearly.

---

## Step 6 — Reporter

Only run this step when `verdict=approved`.

Use the Agent tool to invoke the `reporter` agent:

> Follow the instructions in .claude/agents/reporter/AGENT.md exactly.
> Work on /home/ubuntu/.openclaw/workspace/amz-free-shiping.

Print the PR description it produces directly to the user.

---

## Done

Tell the user:
> "Topology complete. Branch ITZ-54-feat/perf-and-ux-improvements is ready for review."
