#!/usr/bin/env bash
set -Eeuo pipefail

############################
# CONFIG - EDIT THESE ONCE #
############################

# Repo root: auto-detect from script location (works anywhere)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Backend (internal only — frontend proxies to it)
BACKEND_DIR="$ROOT_DIR/backend"
BACKEND_CMD="go run ./cmd/server"
BACKEND_HOST="127.0.0.1"
BACKEND_PORT="8085"

# Frontend
FRONTEND_DIR="$ROOT_DIR/frontend"
FRONTEND_CMD="npm run dev -- -H 127.0.0.1 -p 3000"
FRONTEND_HOST="127.0.0.1"
FRONTEND_PORT="3000"

# Public port exposed on 0.0.0.0 (frontend only)
PUBLIC_FRONTEND_PORT="8002"

# Dependency install commands
FRONTEND_INSTALL_CMD="npm install"
BACKEND_BUILD_CMD=""

# Git pull branch behavior
GIT_PULL_CMD="git pull"

# Logs and pid files
RUN_DIR="$ROOT_DIR/.run"
LOG_DIR="$ROOT_DIR/.logs"

BACKEND_PID_FILE="$RUN_DIR/backend.pid"
FRONTEND_PID_FILE="$RUN_DIR/frontend.pid"
SOCAT_FRONTEND_PID_FILE="$RUN_DIR/socat-frontend.pid"

BACKEND_LOG="$LOG_DIR/backend.log"
FRONTEND_LOG="$LOG_DIR/frontend.log"
SOCAT_FRONTEND_LOG="$LOG_DIR/socat-frontend.log"

#################################
# NO NEED TO EDIT BELOW USUALLY #
#################################

mkdir -p "$RUN_DIR" "$LOG_DIR"

ts() {
  date '+%Y-%m-%d %H:%M:%S'
}

log() {
  echo "[$(ts)] $*"
}

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

kill_pidfile_if_exists() {
  local pidfile="$1"
  local name="$2"

  if [[ -f "$pidfile" ]]; then
    local pid
    pid="$(cat "$pidfile" 2>/dev/null || true)"
    if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null; then
      log "Stopping $name (pid=$pid)"
      kill "$pid" 2>/dev/null || true

      for _ in {1..10}; do
        if kill -0 "$pid" 2>/dev/null; then
          sleep 1
        else
          break
        fi
      done

      if kill -0 "$pid" 2>/dev/null; then
        log "Force killing $name (pid=$pid)"
        kill -9 "$pid" 2>/dev/null || true
      fi
    fi
    rm -f "$pidfile"
  fi
}

kill_port() {
  local port="$1"

  # Try lsof
  if have_cmd lsof; then
    local pids
    pids="$(lsof -ti tcp:"$port" 2>/dev/null || true)"
    if [[ -n "${pids:-}" ]]; then
      log "Killing processes on port $port (lsof): $pids"
      kill $pids 2>/dev/null || true
      sleep 1
      pids="$(lsof -ti tcp:"$port" 2>/dev/null || true)"
      [[ -n "${pids:-}" ]] && kill -9 $pids 2>/dev/null || true
    fi
  fi

  # Try fuser (works better on Linux)
  if have_cmd fuser; then
    if fuser "$port/tcp" >/dev/null 2>&1; then
      log "Killing processes on port $port (fuser)"
      fuser -k "$port/tcp" >/dev/null 2>&1 || true
      sleep 1
      fuser -k -9 "$port/tcp" >/dev/null 2>&1 || true
    fi
  fi

  # Try ss + kill as last resort
  if have_cmd ss; then
    local ss_pids
    ss_pids="$(ss -tlnp "sport = :$port" 2>/dev/null | grep -oP 'pid=\K[0-9]+' || true)"
    if [[ -n "${ss_pids:-}" ]]; then
      log "Killing processes on port $port (ss): $ss_pids"
      kill -9 $ss_pids 2>/dev/null || true
      sleep 1
    fi
  fi

  if ! have_cmd lsof && ! have_cmd fuser && ! have_cmd ss; then
    log "Warning: no lsof/fuser/ss available, cannot auto-kill port $port"
  fi
}

wait_for_port() {
  local host="$1"
  local port="$2"
  local name="$3"
  local timeout="${4:-60}"

  log "Waiting for $name on $host:$port"
  for ((i=1; i<=timeout; i++)); do
    if have_cmd curl; then
      if curl -fsS "http://$host:$port" >/dev/null 2>&1; then
        log "$name is up on $host:$port"
        return 0
      fi
    fi

    if have_cmd ss; then
      if ss -ltn 2>/dev/null | grep -q ":$port "; then
        log "$name is listening on port $port"
        return 0
      fi
    fi

    sleep 1
  done

  log "ERROR: $name did not come up on $host:$port in ${timeout}s"
  return 1
}

start_bg() {
  local workdir="$1"
  local cmd="$2"
  local logfile="$3"
  local pidfile="$4"
  local name="$5"

  log "Starting $name"
  (
    cd "$workdir"
    nohup bash -lc "$cmd" >>"$logfile" 2>&1 &
    echo $! > "$pidfile"
  )

  local pid
  pid="$(cat "$pidfile")"

  sleep 2
  if ! kill -0 "$pid" 2>/dev/null; then
    log "ERROR: $name exited immediately"
    tail -n 50 "$logfile" || true
    return 1
  fi

  log "$name started (pid=$pid)"
}

start_socat_forward() {
  local local_host="$1"
  local local_port="$2"
  local public_port="$3"
  local logfile="$4"
  local pidfile="$5"
  local name="$6"

  kill_pidfile_if_exists "$pidfile" "$name"
  kill_port "$public_port"

  local socat_bin
  socat_bin="$(command -v socat || true)"
  if [[ -z "${socat_bin:-}" ]]; then
    log "ERROR: socat is not installed"
    log "Install with: sudo apt update && sudo apt install -y socat"
    return 1
  fi

  log "Publishing $name: 0.0.0.0:$public_port -> $local_host:$local_port"
  nohup "$socat_bin" \
    TCP-LISTEN:"$public_port",bind=0.0.0.0,fork,reuseaddr \
    TCP:"$local_host":"$local_port" >>"$logfile" 2>&1 &
  echo $! > "$pidfile"

  sleep 2
  local pid
  pid="$(cat "$pidfile")"

  if ! kill -0 "$pid" 2>/dev/null; then
    log "ERROR: socat for $name exited immediately"
    tail -n 50 "$logfile" || true
    return 1
  fi

  log "$name published on 0.0.0.0:$public_port (pid=$pid)"
}

print_summary() {
  echo
  echo "========== RUNNING =========="
  echo "Backend (internal): http://$BACKEND_HOST:$BACKEND_PORT"
  echo "Frontend local:     http://$FRONTEND_HOST:$FRONTEND_PORT"
  echo "Frontend public:    http://0.0.0.0:$PUBLIC_FRONTEND_PORT"
  echo
  echo "Logs:"
  echo "  Backend:          $BACKEND_LOG"
  echo "  Frontend:         $FRONTEND_LOG"
  echo "  Socat frontend:   $SOCAT_FRONTEND_LOG"
  echo
  echo "PID files:"
  echo "  $BACKEND_PID_FILE"
  echo "  $FRONTEND_PID_FILE"
  echo "  $SOCAT_FRONTEND_PID_FILE"
  echo "============================="
  echo
}

show_recent_logs_on_error() {
  echo
  echo "----- backend.log -----"
  tail -n 50 "$BACKEND_LOG" 2>/dev/null || true
  echo
  echo "----- frontend.log -----"
  tail -n 50 "$FRONTEND_LOG" 2>/dev/null || true
  echo
  echo "----- socat-frontend.log -----"
  tail -n 50 "$SOCAT_FRONTEND_LOG" 2>/dev/null || true
}

cleanup_old() {
  log "Stopping old processes"
  kill_pidfile_if_exists "$BACKEND_PID_FILE" "backend"
  kill_pidfile_if_exists "$FRONTEND_PID_FILE" "frontend"
  kill_pidfile_if_exists "$SOCAT_FRONTEND_PID_FILE" "frontend forwarder"

  kill_port "$BACKEND_PORT"
  kill_port "$FRONTEND_PORT"
  kill_port "$PUBLIC_FRONTEND_PORT"

  pkill -f "go run ./cmd/server" 2>/dev/null || true
  pkill -f "socat TCP-LISTEN:$PUBLIC_FRONTEND_PORT" 2>/dev/null || true
}

prepare_code() {
  log "Pulling latest code"
  (
    cd "$ROOT_DIR"
    bash -lc "$GIT_PULL_CMD"
  )

  if [[ -n "$BACKEND_BUILD_CMD" ]]; then
    log "Building backend"
    (
      cd "$BACKEND_DIR"
      bash -lc "$BACKEND_BUILD_CMD"
    )
  fi

  if [[ -f "$FRONTEND_DIR/package.json" ]]; then
    log "Installing frontend dependencies"
    (
      cd "$FRONTEND_DIR"
      bash -lc "$FRONTEND_INSTALL_CMD"
    )
  fi
}

main() {
  trap 'log "ERROR: launch failed"; show_recent_logs_on_error' ERR

  log "Starting auto launch from $ROOT_DIR"
  cleanup_old
  prepare_code

  : > "$BACKEND_LOG"
  : > "$FRONTEND_LOG"
  : > "$SOCAT_FRONTEND_LOG"

  start_bg "$BACKEND_DIR" "$BACKEND_CMD" "$BACKEND_LOG" "$BACKEND_PID_FILE" "backend"
  wait_for_port "$BACKEND_HOST" "$BACKEND_PORT" "backend" 60

  start_bg "$FRONTEND_DIR" "$FRONTEND_CMD" "$FRONTEND_LOG" "$FRONTEND_PID_FILE" "frontend"
  wait_for_port "$FRONTEND_HOST" "$FRONTEND_PORT" "frontend" 60

  start_socat_forward "$FRONTEND_HOST" "$FRONTEND_PORT" "$PUBLIC_FRONTEND_PORT" "$SOCAT_FRONTEND_LOG" "$SOCAT_FRONTEND_PID_FILE" "frontend"

  print_summary
  log "Done"
}

main "$@"
