#!/usr/bin/env python3
"""
CodeRabbit review watcher — polls for a new review without using $() substitution.
Usage: python3 cr-watch.py <repo> <pr_number> <before_review_id> [max_polls] [poll_interval_secs]
Prints: verdict=pr-approved | verdict=new-comments | verdict=triggered-retry
"""
import subprocess, sys, time, json

repo         = sys.argv[1]          # e.g. itzikban/amz-free-shiping
pr           = sys.argv[2]          # e.g. 28
before_id    = sys.argv[3]          # last known review ID as string
max_polls    = int(sys.argv[4]) if len(sys.argv) > 4 else 10
interval     = int(sys.argv[5]) if len(sys.argv) > 5 else 30

def gh_json(args):
    result = subprocess.run(["gh"] + args, capture_output=True, text=True)
    return result.stdout.strip()

def last_cr_review():
    out = gh_json(["api", f"repos/{repo}/pulls/{pr}/reviews",
                   "--jq", '[.[] | select(.user.login == "coderabbitai[bot]")] | last | {id: .id, state: .state}'])
    if not out:
        return None, None
    try:
        data = json.loads(out)
        return str(data.get("id", "")), data.get("state", "")
    except Exception:
        return None, None

def new_comment_count(since_id):
    out = gh_json(["api", f"repos/{repo}/pulls/{pr}/comments",
                   "--jq", f'[.[] | select(.user.login == "coderabbitai[bot]") | select(.id > {since_id})] | length'])
    try:
        return int(out)
    except Exception:
        return 0

print(f"Watching PR #{pr} on {repo} (before_id={before_id})", flush=True)

for i in range(1, max_polls + 1):
    review_id, state = last_cr_review()
    print(f"Poll {i}/{max_polls}: review_id={review_id} state={state}", flush=True)

    if review_id and review_id != before_id:
        print(f"New review detected: id={review_id} state={state}", flush=True)
        count = new_comment_count(before_id)
        print(f"New inline comments since before_id: {count}", flush=True)
        if state == "APPROVED":
            print("verdict=pr-approved")
        elif state == "CHANGES_REQUESTED":
            print("verdict=new-comments")
        elif state == "COMMENTED":
            print("verdict=pr-approved" if count == 0 else "verdict=new-comments")
        else:
            print("verdict=new-comments")
        sys.exit(0)

    if i < max_polls:
        time.sleep(interval)

# No new review — push empty commit to trigger
print("No new review after polling. Pushing trigger commit...", flush=True)
subprocess.run(["git", "commit", "--allow-empty", "-m", "chore: trigger CodeRabbit re-review [skip ci]"])
subprocess.run(["git", "push"])
print("verdict=triggered-retry")
