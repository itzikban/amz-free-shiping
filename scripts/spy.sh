#!/usr/bin/env bash
# spy.sh — watch the mission-loop cron in real time

export TERM=xterm-256color

REPO="/home/ubuntu/.openclaw/workspace/amz-free-shiping"
JOBS_FILE="$HOME/.openclaw/cron/jobs.json"
CRON_DIR="$HOME/.openclaw/cron/runs"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BLUE='\033[0;34m'; BOLD='\033[1m'; RESET='\033[0m'

clear
echo -e "${BOLD}╔══════════════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}║          🕵️  OpenClaw Mission Spy                ║${RESET}"
echo -e "${BOLD}╚══════════════════════════════════════════════════╝${RESET}"
echo ""

# ── find enabled mission-loop job ────────────────────────────────────────────
JOB_ID=$(python3 -c "
import json
jobs = json.load(open('$JOBS_FILE'))['jobs']
for j in jobs:
    if j.get('enabled') and 'mission' in j.get('name','').lower():
        print(j['id']); break
" 2>/dev/null)

[[ -z "$JOB_ID" ]] && JOB_ID="3fc862e5-0998-4b57-9730-1dd52ddae18c"

CRON_LOG="$CRON_DIR/$JOB_ID.jsonl"
touch "$CRON_LOG" 2>/dev/null || true

echo -e "${CYAN}Job  :${RESET} $JOB_ID"
echo -e "${CYAN}Log  :${RESET} $CRON_LOG"
echo ""

section() { echo -e "\n${BOLD}${BLUE}── $* ──${RESET}"; }

# ── initial snapshot ──────────────────────────────────────────────────────────
section "GIT (last 5 commits)"
git -C "$REPO" log --oneline -5 2>/dev/null

section "OPEN PRs"
gh pr list --state open --json number,title,headRefName \
  --template '{{range .}}  #{{.number}}  {{.headRefName}}  —  {{.title}}{{"\n"}}{{end}}' 2>/dev/null

section "SERVICES"
curl -fs http://127.0.0.1:8085/health    >/dev/null 2>&1 && echo -e "  Backend  ${GREEN}● UP${RESET}" || echo -e "  Backend  ${RED}● DOWN${RESET}"
curl -fs http://127.0.0.1:8002/api/health >/dev/null 2>&1 && echo -e "  Frontend ${GREEN}● UP${RESET}" || echo -e "  Frontend ${RED}● DOWN${RESET}"

section "AGENT PROCESSES"
ps aux 2>/dev/null | grep -E 'openclaw-gateway|claw-worker' | grep -v grep \
  | awk '{printf "  pid=%-7s cpu=%-5s mem=%-5s %s\n", $2,$3,$4,substr($0,index($0,$11))}'

section "LAST CRON RUN"
python3 -c "
import json, datetime
jobs = json.load(open('$JOBS_FILE'))['jobs']
for j in jobs:
    if j['id'] == '$JOB_ID':
        s = j.get('state', {})
        ms = s.get('lastRunAtMs') or s.get('nextRunAtMs')
        if ms:
            dt = datetime.datetime.fromtimestamp(ms/1000).strftime('%Y-%m-%d %H:%M:%S')
            label = 'Last run' if s.get('lastRunAtMs') else 'Next run'
            print(f'  {label}: {dt}  status={s.get(\"lastRunStatus\",\"?\")}'  )
        else:
            print('  No run recorded yet')
        break
" 2>/dev/null

section "CRON LOG (live)"
echo -e "  ${YELLOW}Showing last 5 runs + streaming new cycles...${RESET}"
echo ""

# ── background watchers ───────────────────────────────────────────────────────
(
  last=$(git -C "$REPO" log --oneline -1 2>/dev/null)
  while true; do
    cur=$(git -C "$REPO" log --oneline -1 2>/dev/null)
    if [[ "$cur" != "$last" && -n "$last" ]]; then
      ts=$(date '+%H:%M:%S')
      echo -e "\n${GREEN}${BOLD}[$ts] 🚀 NEW COMMIT!${RESET}  $cur"
    fi
    last="$cur"; sleep 5
  done
) &
GIT_PID=$!

(
  lb=""; lf=""
  while true; do
    b=$(curl -fs http://127.0.0.1:8085/health    >/dev/null 2>&1 && echo up || echo down)
    f=$(curl -fs http://127.0.0.1:8002/api/health >/dev/null 2>&1 && echo up || echo down)
    ts=$(date '+%H:%M:%S')
    [[ "$b" != "$lb" && -n "$lb" ]] && { [[ "$b" == up ]] \
      && echo -e "\n${GREEN}${BOLD}[$ts] ✅ Backend UP${RESET}" \
      || echo -e "\n${RED}${BOLD}[$ts] 💀 Backend DOWN${RESET}"; }
    [[ "$f" != "$lf" && -n "$lf" ]] && { [[ "$f" == up ]] \
      && echo -e "\n${GREEN}${BOLD}[$ts] ✅ Frontend UP${RESET}" \
      || echo -e "\n${RED}${BOLD}[$ts] 💀 Frontend DOWN${RESET}"; }
    lb="$b"; lf="$f"; sleep 10
  done
) &
SVC_PID=$!

trap "kill $GIT_PID $SVC_PID 2>/dev/null; echo ''; echo 'spy stopped.'; exit 0" INT TERM EXIT

# ── Python log poller ─────────────────────────────────────────────────────────
python3 -u << PYEOF
import time, json, os, sys

LOG   = "$CRON_LOG"
RESET = "\033[0m"; BOLD = "\033[1m"

def fmt_entry(e):
    try:
        ts_ms   = int(e.get("ts") or e.get("runAtMs") or 0)
        ts_str  = time.strftime('%H:%M:%S', time.localtime(ts_ms / 1000)) if ts_ms else "??:??:??"
        action  = str(e.get("action") or "")
        status  = str(e.get("status") or "")
        dur     = int(e.get("durationMs") or 0)
        model   = str(e.get("model") or "")
        usage   = e.get("usage") if isinstance(e.get("usage"), dict) else {}
        tokens  = int(usage.get("total_tokens") or 0)
        summary = str(e.get("summary") or "")

        if action == "finished":
            sc    = "\033[0;32m" if status == "ok" else "\033[0;31m"
            icon  = "✅" if status == "ok" else "❌"
            dur_s = f"{dur/1000:.1f}s" if dur else ""
            tok_s = f"  {tokens:,} tokens" if tokens else ""
            mod_s = f"  [{model}]" if model else ""
            print(f"\n  {BOLD}{sc}[{ts_str}] {icon} RUN {status.upper()}{RESET}{sc}  {dur_s}{tok_s}{mod_s}{RESET}")
            if summary:
                for line in summary.strip().splitlines():
                    if line.strip():
                        col = "\033[0;31m" if any(w in line.upper() for w in ["BLOCKER","ERROR","FAIL","HUMAN-NEEDED","CONFLICT"]) \
                              else "\033[0;32m" if any(w in line.upper() for w in ["MERGE","DONE","PASSED","PUSHED","POSTED"]) \
                              else "\033[0;37m"
                        print(f"    {col}{line}{RESET}")
        elif action == "started":
            print(f"  \033[0;36m[{ts_str}] 🔄 cycle started{RESET}")
        else:
            print(f"  [{ts_str}] {action}: {str(e)[:160]}")

    except Exception as ex:
        print(f"  [parse-err] {ex} — {str(e)[:160]}")

    sys.stdout.flush()

# show last 5 runs from existing log
with open(LOG, "r", errors="replace") as fh:
    existing = [l.strip() for l in fh if l.strip()]
for raw in existing[-5:]:
    try:
        fmt_entry(json.loads(raw))
    except Exception:
        print(f"  {raw[:160]}")

print(f"\n  \033[0;33m── streaming new cycles below ──\033[0m\n", flush=True)

pos = os.path.getsize(LOG)
last_heartbeat = time.time()
last_cron_size = pos

while True:
    try:
        size = os.path.getsize(LOG)
    except FileNotFoundError:
        time.sleep(0.5)
        continue

    if size > pos:
        with open(LOG, "r", errors="replace") as fh:
            fh.seek(pos)
            for raw in fh:
                raw = raw.strip()
                if not raw:
                    continue
                try:
                    fmt_entry(json.loads(raw))
                except Exception:
                    print(f"  {raw[:200]}")
                    sys.stdout.flush()
        pos = size

    now = time.time()
    if now - last_heartbeat >= 30:
        ts_s = time.strftime('%H:%M:%S')
        if pos == last_cron_size:
            print(f"  \033[0;33m[{ts_s}] ⏳ idle — watching for next cron cycle\033[0m", flush=True)
        last_heartbeat = now
        last_cron_size = pos

    time.sleep(0.5)
PYEOF
