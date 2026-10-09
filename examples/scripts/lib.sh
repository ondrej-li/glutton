#!/usr/bin/env bash
# Shared helpers for the glutton example scripts. Source it, do not run it.
#
#   source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WORK_DIR="${GLUTTON_EXAMPLES_DIR:-/tmp/glutton-examples}"
GLUTTON_BINARY="${GLUTTON_BINARY:-$WORK_DIR/glutton}"
GLUTTON_LOG="$WORK_DIR/glutton.log"
GLUTTON_PID=""

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
pass() { printf '\033[1;32m ok \033[0m%s\n' "$*"; }
skip() { printf '\033[1;33mskip \033[0m%s\n' "$*"; }
fail() {
  printf '\033[1;31mfail \033[0m%s\n' "$*" >&2
  exit 1
}

# build_glutton compiles the binary used by the examples into $GLUTTON_BINARY.
build_glutton() {
  mkdir -p "$WORK_DIR"
  log "building glutton into $GLUTTON_BINARY"
  (cd "$REPO_ROOT" && go build -o "$GLUTTON_BINARY" ./cmd/glutton)
}

# start_glutton <config-file|-> [extra flags...] starts glutton in the background.
# Pass "-" as the config to configure the single route purely from environment variables.
# The server is always stopped again on exit.
start_glutton() {
  local config="${1:-}"
  [ -n "$config" ] && shift
  mkdir -p "$WORK_DIR"
  if [ -n "$config" ] && [ "$config" != "-" ]; then
    log "starting glutton with $config"
    "$GLUTTON_BINARY" -f "$config" "$@" >"$GLUTTON_LOG" 2>&1 &
  else
    log "starting glutton with an environment based configuration"
    "$GLUTTON_BINARY" "$@" >"$GLUTTON_LOG" 2>&1 &
  fi
  GLUTTON_PID=$!
  trap stop_glutton EXIT
}

# stop_glutton asks the server to shut down gracefully (SIGINT) and waits for it.
stop_glutton() {
  if [ -n "$GLUTTON_PID" ] && kill -0 "$GLUTTON_PID" 2>/dev/null; then
    log "stopping glutton ($GLUTTON_PID)"
    kill -INT "$GLUTTON_PID" 2>/dev/null || true
    for _ in $(seq 1 50); do
      kill -0 "$GLUTTON_PID" 2>/dev/null || break
      sleep 0.1
    done
    if kill -0 "$GLUTTON_PID" 2>/dev/null; then
      kill "$GLUTTON_PID" 2>/dev/null || true
    fi
    wait "$GLUTTON_PID" 2>/dev/null || true
  fi
  GLUTTON_PID=""
}

# wait_for_glutton <url> [curl args...] polls until the server (or its TLS handshake) answers.
wait_for_glutton() {
  local url="$1"
  shift
  for _ in $(seq 1 50); do
    if curl -s -o /dev/null "$@" "$url" 2>/dev/null; then
      pass "glutton is reachable at $url"
      return 0
    fi
    if [ -n "$GLUTTON_PID" ] && ! kill -0 "$GLUTTON_PID" 2>/dev/null; then
      cat "$GLUTTON_LOG" >&2
      fail "glutton exited during startup, see $GLUTTON_LOG"
    fi
    sleep 0.1
  done
  cat "$GLUTTON_LOG" >&2
  fail "glutton did not become reachable at $url"
}

# expect_status <expected> <url> [curl args...] fails unless curl returns the expected status.
expect_status() {
  local expected="$1"
  local url="$2"
  shift 2
  local status
  status="$(curl -s -o /dev/null -w '%{http_code}' "$@" "$url")"
  [ "$status" = "$expected" ] || fail "expected HTTP $expected from $url but got $status"
  pass "HTTP $status $url"
}
