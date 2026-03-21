---
name: pr-reader
description: "For each PR in the list, checks out its branch and fetches ALL unresolved CodeRabbit comments across every file. Filters out already-fixed comments using the local cache."
model: haiku
tools:
  - Bash
  - Read
---

## Instructions

You receive a list of PR numbers. Process them one at a time.

**Step 0 — Load the fixed-comment cache:**

Use the Read tool to read the cache file at:
  /home/ubuntu/.openclaw/workspace/amz-free-shiping/.coderabbit-cache/fixed.json

Note the `fixed_comment_ids` array. Any comment whose `id` appears in this list has already been fixed — skip it.

If the file doesn't exist or is empty, treat the fixed list as empty.

**Step 1 — Get repo (one simple Bash call, note the output):**

  gh repo view --json owner,name --jq '"\(.owner.login)/\(.name)"'

**For each PR number:**

**Step 2 — Check out the PR branch:**

  gh pr checkout <PR_NUMBER>

**Step 3 — Fetch ALL CodeRabbit inline comments (substitute REPO and PR_NUMBER literally):**

  gh api repos/REPO/pulls/PR_NUMBER/comments --paginate --jq '[.[] | select(.user.login == "coderabbitai[bot]") | {id: .id, path: .path, line: .line, body: .body, created_at: .created_at}]'

**Step 4 — Fetch review-level comments:**

  gh api repos/REPO/pulls/PR_NUMBER/reviews --paginate --jq '[.[] | select(.user.login == "coderabbitai[bot]") | {id: .id, body: .body, state: .state}]'

**Step 5 — Filter and print:**

For each comment, check if its `id` is in the `fixed_comment_ids` list from the cache.
- If YES → skip it (already fixed in a previous round).
- If NO → print it in this format:

  ID: <comment id>
  PR: <number>
  BRANCH: <branch name>
  FILE: <path>
  LINE: <line number or "general">
  ISSUE: <full comment body>
  ---

If all comments for a PR are already cached, skip the PR entirely.

**After processing all PRs:**
- If at least one new (uncached) comment was found → output status=has-comments
- If all comments are already in the cache → output status=clean

## Outputs
- status: has-comments | clean
