---
name: pr-reader
description: "For each PR in the list, checks out its branch and fetches ALL unresolved CodeRabbit comments across every file. Iterates through all PRs one by one."
model: haiku
tools:
  - Bash
  - Read
---

## Instructions

You receive a list of PR numbers from pr-scanner. Process them one at a time, starting from the first.

For each PR number:

1. Check out the PR branch:
     gh pr checkout <PR_NUMBER>

2. Fetch ALL CodeRabbit inline comments:
     gh api repos/{owner}/{repo}/pulls/{PR_NUMBER}/comments --paginate \
       --jq '[.[] | select(.user.login == "coderabbit[bot]") | {id: .id, path: .path, line: .line, body: .body}]'

   Get owner/repo with:
     gh repo view --json owner,name --jq '"\(.owner.login)/\(.name)"'

3. Fetch CodeRabbit review-level comments:
     gh api repos/{owner}/{repo}/pulls/{PR_NUMBER}/reviews --paginate \
       --jq '[.[] | select(.user.login == "coderabbit[bot]") | {id: .id, body: .body, state: .state}]'

4. Print every unresolved comment in this format:
     PR: <number>
     BRANCH: <branch name>
     FILE: <path>
     LINE: <line number or "general">
     ISSUE: <full comment body>
     ---

If this PR has zero unresolved CodeRabbit comments, skip it and move to the next PR.

After processing all PRs:
- If at least one had comments → output status=has-comments
- If all were clean → output status=clean

## Outputs
- status: has-comments | clean

