#!/usr/bin/env python3
import json
import os
import subprocess
import sys
from datetime import datetime, timezone

REPO_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
STATE_PATH = os.path.join(REPO_DIR, ".coderabbit-watch-state.json")
LOG_PATH = os.path.join(REPO_DIR, ".coderabbit-watch.log")


def run(cmd):
    return subprocess.check_output(cmd, cwd=REPO_DIR, text=True).strip()


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


def main():
    try:
        prs_raw = run([
            "gh",
            "pr",
            "list",
            "--state",
            "open",
            "--json",
            "number,headRefOid,isDraft,title,url",
        ])
        prs = json.loads(prs_raw)
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

        prev = state["prs"].get(num, {})
        prev_sha = prev.get("last_triggered_sha")

        if not sha:
            continue

        if prev_sha != sha:
            try:
                body = "@coderabbitai review\n\nAuto-triggered by watcher: new commit detected on this PR."
                run(["gh", "pr", "comment", num, "--body", body])
                state["prs"][num] = {
                    "last_triggered_sha": sha,
                    "last_triggered_at": datetime.now(timezone.utc).isoformat(),
                    "title": pr.get("title", ""),
                    "url": pr.get("url", ""),
                }
                triggered.append(f"#{num}")
            except Exception as e:
                log(f"ERROR triggering PR #{num}: {e}")

    # prune closed PR state entries
    open_nums = {str(pr["number"]) for pr in prs}
    for k in list(state["prs"].keys()):
        if k not in open_nums:
            del state["prs"][k]

    save_state(state)

    if triggered:
        log(f"Triggered CodeRabbit review for: {', '.join(triggered)}")
    else:
        log("No new commits on open PRs; no trigger needed")

    # Operator reminder for assistant workflow
    reminder = (
        "REMINDER: After resolving CodeRabbit comments, summarize resolved/remaining "
        "items to the user and explicitly ask whether to merge the PR."
    )
    log(reminder)
    print(reminder)

    return 0


if __name__ == "__main__":
    sys.exit(main())
