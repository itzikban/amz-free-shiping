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

        # Track latest CodeRabbit review and actionable count.
        try:
            rev = latest_coderabbit_review(num)
            if rev and rev.get("id"):
                actionable = parse_actionable_count(rev.get("body", ""))
                prev_rev_id = prev.get("last_coderabbit_review_id")
                if prev_rev_id != rev["id"]:
                    msg = f"PR #{num} CodeRabbit review updated"
                    if actionable is not None:
                        msg += f" (actionable={actionable})"
                    log(msg)

                state["prs"][num] = {
                    **state["prs"].get(num, {}),
                    "last_coderabbit_review_id": rev["id"],
                    "last_coderabbit_review_at": rev.get("submitted_at"),
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
