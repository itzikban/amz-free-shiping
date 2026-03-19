#!/usr/bin/env python3
"""
Babysitter + Watchdog (repo-local)

Purpose:
- Keep branch work aligned to a declared ticket scope.
- Detect inactivity/progress drift.
- Emit concise machine-readable status for cron integrations.

This script intentionally does NOT modify app code.
"""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path

REPO = Path(__file__).resolve().parents[1]
STATE_DIR = REPO / ".agent"
STATE_FILE = STATE_DIR / "watchdog-state.json"
MAP_FILE = STATE_DIR / "branch-map.json"  # local-only, gitignored

INACTIVITY_SECONDS = int(os.environ.get("BABYSITTER_INACTIVITY_SECONDS", "2700"))  # 45m


@dataclass
class Result:
    status: str
    summary: str
    branch: str
    head: str
    ticket: str | None
    stalled: bool


def run(*cmd: str) -> str:
    return subprocess.check_output(cmd, cwd=str(REPO), text=True).strip()


def git_branch() -> str:
    return run("git", "branch", "--show-current")


def git_head() -> str:
    return run("git", "rev-parse", "--short", "HEAD")


def git_last_commit_ts() -> int:
    return int(run("git", "log", "-1", "--format=%ct"))


def infer_ticket_from_branch(branch: str) -> str | None:
    m = re.search(r"\b([A-Z]{2,10}-\d+)\b", branch)
    return m.group(1) if m else None


def read_branch_map() -> dict:
    if not MAP_FILE.exists():
        return {}
    try:
        return json.loads(MAP_FILE.read_text(encoding="utf-8"))
    except Exception:
        return {}


def load_state() -> dict:
    if not STATE_FILE.exists():
        return {}
    try:
        return json.loads(STATE_FILE.read_text(encoding="utf-8"))
    except Exception:
        return {}


def save_state(state: dict) -> None:
    STATE_DIR.mkdir(parents=True, exist_ok=True)
    STATE_FILE.write_text(json.dumps(state, indent=2), encoding="utf-8")


def main() -> int:
    branch = git_branch()
    head = git_head()
    last_commit_ts = git_last_commit_ts()
    now = int(time.time())

    bmap = read_branch_map()
    mapped_ticket = bmap.get(branch)
    inferred_ticket = infer_ticket_from_branch(branch)
    ticket = mapped_ticket or inferred_ticket

    state = load_state()
    prev = state.get(branch, {})

    head_changed = prev.get("head") != head
    if head_changed:
        prev["last_progress_ts"] = now
    elif "last_progress_ts" not in prev:
        prev["last_progress_ts"] = last_commit_ts

    inactive_for = now - int(prev.get("last_progress_ts", last_commit_ts))
    stalled = inactive_for >= INACTIVITY_SECONDS

    if not ticket:
        status = "WARN"
        summary = "No ticket linked to branch (set .agent/branch-map.json or include ITZ-### in branch name)."
    elif stalled:
        status = "STALE"
        summary = f"No new commit for {inactive_for // 60}m on {branch}."
    else:
        status = "OK"
        summary = f"Branch aligned: {branch} -> {ticket}. Last progress {inactive_for // 60}m ago."

    prev.update(
        {
            "head": head,
            "ticket": ticket,
            "branch": branch,
            "last_commit_ts": last_commit_ts,
            "last_checked_ts": now,
        }
    )
    state[branch] = prev
    save_state(state)

    result = Result(status, summary, branch, head, ticket, stalled)
    print(json.dumps(result.__dict__, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main())
