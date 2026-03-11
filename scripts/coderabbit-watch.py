#!/usr/bin/env python3
import json
import os
import re
import subprocess
import sys
from datetime import datetime, timezone

REPO_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
STATE_PATH = os.path.join(REPO_DIR, ".coderabbit-watch-state.json")
LOG_PATH = os.path.join(REPO_DIR, ".coderabbit-watch.log")


def run(cmd):
    return subprocess.check_output(cmd, cwd=REPO_DIR, text=True).strip()


def run_json(cmd):
    out = run(cmd)
    return json.loads(out) if out else {}


def load_state():
    if not os.path.exists(STATE_PATH):
        return {"prs": {}}
    with open(STATE_PATH, "r", encoding="utf-8") as f:
        return json.load(f)


def save_state(state):
    with open(STATE_PATH, "w", encoding="utf-8") as f:
        json.dump(state, f, indent=2)


def log(msg):
    ts = datetime.now(timezone.utc).isoformat()
    with open(LOG_PATH, "a", encoding="utf-8") as f:
        f.write(f"[{ts}] {msg}\n")


def latest_coderabbit_review(pr_number: str):
    reviews = run_json(["gh", "api", f"/repos/itzikban/amz-free-shiping/pulls/{pr_number}/reviews"])
    for r in reversed(reviews):
        user = (r.get("user") or {}).get("login", "").lower()
        body = (r.get("body") or "")
        if "coderabbit" in user or "coderabbit" in body.lower():
            return {
                "id": r.get("id"),
                "submitted_at": r.get("submitted_at"),
                "body": body,
            }
    return None


def latest_coderabbit_issue_comment(pr_number: str):
    comments = run_json(["gh", "api", f"/repos/itzikban/amz-free-shiping/issues/{pr_number}/comments"])
    for c in reversed(comments):
        user = (c.get("user") or {}).get("login", "").lower()
        body = (c.get("body") or "")
        if "coderabbit" in user or "coderabbit" in body.lower():
            return {
                "id": c.get("id"),
                "created_at": c.get("created_at"),
                "body": body,
            }
    return None


def coderabbit_inline_comments(pr_number: str):
    comments = run_json(["gh", "api", f"/repos/itzikban/amz-free-shiping/pulls/{pr_number}/comments"])
    out = []
    for c in comments:
        user = (c.get("user") or {}).get("login", "").lower()
        if "coderabbit" not in user:
            continue
        out.append({
            "id": c.get("id"),
            "path": c.get("path"),
            "line": c.get("line"),
            "created_at": c.get("created_at"),
            "body": c.get("body") or "",
        })
    return out


def parse_iso(ts: str):
    if not ts:
        return None
    return datetime.fromisoformat(ts.replace("Z", "+00:00"))


def parse_actionable_count(review_body: str):
    m = re.search(r"Actionable comments posted:\s*(\d+)", review_body or "", flags=re.I)
    if not m:
        return None
    return int(m.group(1))


def parse_confirmed_fixed_count(issue_comment_body: str):
    # CodeRabbit often writes: "All N previously confirmed fixes remain in place"
    m = re.search(r"all\s+(\d+)\s+previously\s+confirmed\s+fixes", issue_comment_body or "", flags=re.I)
    if not m:
        return None
    return int(m.group(1))


REVIEW_RESPONSE_WAIT_SECONDS = 120
ESCALATION_SECONDS = 300


def main():
    try:
        prs = run_json([
            "gh", "pr", "list", "--state", "open",
            "--json", "number,headRefOid,isDraft,title,url",
        ])
    except Exception as e:
        log(f"ERROR listing PRs: {e}")
        return 1

    state = load_state()
    state.setdefault("prs", {})

    for pr in prs:
        if pr.get("isDraft"):
            continue

        num = str(pr["number"])
        sha = pr.get("headRefOid") or ""
        if not sha:
            continue

        prev = state["prs"].get(num, {})

        # Gather latest CodeRabbit signals
        try:
            rev = latest_coderabbit_review(num)
            cm = latest_coderabbit_issue_comment(num)
            inline = coderabbit_inline_comments(num)
        except Exception as e:
            log(f"ERROR fetching CodeRabbit data PR #{num}: {e}")
            continue

        rev_id = (rev or {}).get("id")
        cm_id = (cm or {}).get("id")
        actionable = parse_actionable_count((rev or {}).get("body", "")) if rev else None
        confirmed_fixed = parse_confirmed_fixed_count((cm or {}).get("body", "")) if cm else None

        # Inline fingerprint: detect unseen bot inline comments (might be missed by summary)
        inline_ids = [x["id"] for x in inline]
        inline_fp = f"{len(inline_ids)}:{inline_ids[-1] if inline_ids else 0}"

        changed_activity = (
            prev.get("last_coderabbit_review_id") != rev_id
            or prev.get("last_coderabbit_comment_id") != cm_id
            or prev.get("last_inline_fingerprint") != inline_fp
        )

        if changed_activity:
            log(f"PR #{num} CodeRabbit activity updated (actionable={actionable}, inline={len(inline_ids)})")
            prev["pending_since"] = None
            prev["retrigger_count"] = 0
            prev["escalated"] = False

        # SHA-gated retrigger: only retrigger if current SHA wasn't requested yet
        last_requested_sha = prev.get("last_requested_review_sha")
        now = datetime.now(timezone.utc)

        # determine if still pending unresolved work
        pending_unresolved = False
        if actionable is not None and actionable > 0:
            pending_unresolved = True
        elif actionable is None and inline_ids:
            # fallback: if no actionable summary but inline comments exist, treat as pending
            pending_unresolved = True

        if pending_unresolved:
            if not prev.get("pending_since"):
                prev["pending_since"] = now.isoformat()
            pending_since = parse_iso(prev.get("pending_since")) or now
            elapsed = (now - pending_since).total_seconds()
            retrigger_count = int(prev.get("retrigger_count") or 0)

            if elapsed >= 120 and retrigger_count < 1 and last_requested_sha != sha:
                try:
                    body = (
                        "@coderabbitai review\n\n"
                        "Auto-retrigger (SHA-gated): unresolved findings detected and this commit SHA has not been requested yet."
                    )
                    run(["gh", "pr", "comment", num, "--body", body])
                    prev["retrigger_count"] = retrigger_count + 1
                    prev["last_requested_review_sha"] = sha
                    prev["last_requested_review_at"] = now.isoformat()
                    log(f"PR #{num} retriggered CodeRabbit on new SHA {sha[:12]}")
                except Exception as e:
                    log(f"ERROR retriggering PR #{num}: {e}")

            if elapsed >= 300 and not prev.get("escalated"):
                log(f"ESCALATION PR #{num}: unresolved findings for 5+ minutes (actionable={actionable}, inline={len(inline_ids)})")
                prev["escalated"] = True
        else:
            prev["pending_since"] = None
            prev["retrigger_count"] = 0
            prev["escalated"] = False

        prev.update({
            "title": pr.get("title", ""),
            "url": pr.get("url", ""),
            "last_head_sha": sha,
            "last_coderabbit_review_id": rev_id,
            "last_coderabbit_review_at": (rev or {}).get("submitted_at"),
            "last_coderabbit_actionable": actionable,
            "last_coderabbit_comment_id": cm_id,
            "last_coderabbit_comment_at": (cm or {}).get("created_at"),
            "last_coderabbit_confirmed_fixed": confirmed_fixed,
            "last_inline_fingerprint": inline_fp,
            "last_inline_count": len(inline_ids),
        })
        state["prs"][num] = prev

        if actionable == 0:
            log(
                f"MERGE-ASK REMINDER PR #{num}: actionable=0. Summarize resolved/remaining and ask user if they want merge."
            )

    open_nums = {str(pr["number"]) for pr in prs}
    for k in list(state["prs"].keys()):
        if k not in open_nums:
            del state["prs"][k]

    save_state(state)

    reminder = (
        "REMINDER: After resolving CodeRabbit comments, summarize resolved/remaining "
        "items to the user and explicitly ask whether to merge the PR."
    )
    log(reminder)
    print(reminder)
    return 0


if __name__ == "__main__":
    sys.exit(main())
