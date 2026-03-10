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
    line = f"[{ts}] {msg}\n"
    with open(LOG_PATH, "a", encoding="utf-8") as f:
        f.write(line)


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


def parse_iso(ts: str):
    if not ts:
        return None
    return datetime.fromisoformat(ts.replace("Z", "+00:00"))


def parse_actionable_count(review_body: str):
    m = re.search(r"Actionable comments posted:\s*(\d+)", review_body or "", flags=re.I)
    if not m:
        return None
    return int(m.group(1))


def main():
    try:
        prs = run_json([
            "gh",
            "pr",
            "list",
            "--state",
            "open",
            "--json",
            "number,headRefOid,isDraft,title,url",
        ])
    except Exception as e:
        log(f"ERROR listing PRs: {e}")
        return 1

    state = load_state()
    state.setdefault("prs", {})

    triggered = []
    for pr in prs:
        if pr.get("isDraft"):
            continue

        num = str(pr["number"])
        sha = pr.get("headRefOid") or ""
        if not sha:
            continue

        prev = state["prs"].get(num, {})
        prev_sha = prev.get("last_triggered_sha")

        # Trigger CodeRabbit only when commit SHA changes.
        if prev_sha != sha:
            try:
                body = "@coderabbitai review\n\nAuto-triggered by watcher: new commit detected on this PR."
                run(["gh", "pr", "comment", num, "--body", body])
                state["prs"][num] = {
                    **prev,
                    "last_triggered_sha": sha,
                    "last_triggered_at": datetime.now(timezone.utc).isoformat(),
                    "title": pr.get("title", ""),
                    "url": pr.get("url", ""),
                }
                triggered.append(f"#{num}")
            except Exception as e:
                log(f"ERROR triggering PR #{num}: {e}")

        # Track latest CodeRabbit review/comment activity + actionable count.
        try:
            rev = latest_coderabbit_review(num)
            cm = latest_coderabbit_issue_comment(num)

            actionable = parse_actionable_count((rev or {}).get("body", "")) if rev else None
            rev_id = (rev or {}).get("id")
            cm_id = (cm or {}).get("id")
            rev_at = parse_iso((rev or {}).get("submitted_at"))
            cm_at = parse_iso((cm or {}).get("created_at"))

            latest_at = rev_at
            if cm_at and (latest_at is None or cm_at > latest_at):
                latest_at = cm_at

            prev_state = state["prs"].get(num, {})
            changed = (prev_state.get("last_coderabbit_review_id") != rev_id) or (prev_state.get("last_coderabbit_comment_id") != cm_id)

            if changed:
                msg = f"PR #{num} CodeRabbit activity updated"
                if actionable is not None:
                    msg += f" (actionable={actionable})"
                log(msg)
                prev_state["pending_since"] = None
                prev_state["retrigger_count"] = 0
                prev_state["escalated"] = False

            # Pending logic when actionable unresolved and no fresh activity
            now = datetime.now(timezone.utc)
            if actionable is not None and actionable > 0:
                if not prev_state.get("pending_since"):
                    prev_state["pending_since"] = now.isoformat()
                pending_since = parse_iso(prev_state.get("pending_since")) or now
                elapsed = (now - pending_since).total_seconds()
                retrigger_count = int(prev_state.get("retrigger_count") or 0)

                if elapsed >= 120 and retrigger_count < 1:
                    body = "@coderabbitai review\n\nAuto-retrigger: no new CodeRabbit review activity for 2+ minutes while actionable comments remain."
                    run(["gh", "pr", "comment", num, "--body", body])
                    prev_state["retrigger_count"] = retrigger_count + 1
                    log(f"PR #{num} auto-retriggered CodeRabbit review (count={retrigger_count + 1})")

                if elapsed >= 300 and not prev_state.get("escalated"):
                    log(f"ESCALATION PR #{num}: no fresh CodeRabbit activity for 5+ minutes while actionable comments remain")
                    prev_state["escalated"] = True
            else:
                prev_state["pending_since"] = None
                prev_state["retrigger_count"] = 0
                prev_state["escalated"] = False

            state["prs"][num] = {
                **prev_state,
                "last_coderabbit_review_id": rev_id,
                "last_coderabbit_review_at": (rev or {}).get("submitted_at"),
                "last_coderabbit_comment_id": cm_id,
                "last_coderabbit_comment_at": (cm or {}).get("created_at"),
                "last_coderabbit_activity_at": latest_at.isoformat() if latest_at else None,
                "last_coderabbit_actionable": actionable,
                "title": pr.get("title", ""),
                "url": pr.get("url", ""),
            }

            # Reminder when review appears clean
            if actionable == 0:
                log(
                    f"MERGE-ASK REMINDER PR #{num}: CodeRabbit actionable comments are 0. "
                    "Summarize resolved/remaining and ask user if they want to merge."
                )
        except Exception as e:
            log(f"ERROR reading CodeRabbit state for PR #{num}: {e}")

    # Prune closed PR state
    open_nums = {str(pr["number"]) for pr in prs}
    for k in list(state["prs"].keys()):
        if k not in open_nums:
            del state["prs"][k]

    save_state(state)

    if triggered:
        log(f"Triggered CodeRabbit review for: {', '.join(triggered)}")
    else:
        log("No new commits on open PRs; no trigger needed")

    reminder = (
        "REMINDER: After resolving CodeRabbit comments, summarize resolved/remaining "
        "items to the user and explicitly ask whether to merge the PR."
    )
    log(reminder)
    print(reminder)
    return 0


if __name__ == "__main__":
    sys.exit(main())
