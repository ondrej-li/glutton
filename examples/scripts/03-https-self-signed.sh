#!/usr/bin/env bash
# Example 3: HTTPS using a certificate generated in memory at startup.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

OUTPUT=/tmp/glutton-examples/tls-self-signed
URL=https://127.0.0.1:4356/v1/glutton/save

build_glutton
rm -rf "$OUTPUT"
start_glutton "$REPO_ROOT/examples/configs/tls-self-signed.yaml"
wait_for_glutton "$URL" -k

log "the startup log says which names the generated certificate covers:"
grep -o "using a generated self signed one valid for .*" "$GLUTTON_LOG" || true

log "without -k the generated certificate is not trusted"
if curl -s -o /dev/null "$URL" 2>/dev/null; then
  fail "expected certificate verification to fail"
fi
pass "certificate verification fails as expected"

log "posting a payload with -k (skip verification)"
expect_status 200 "$URL" -k -X POST -d '{"hello":"glutton"}'

log "payloads stored in $OUTPUT:"
ls -1 "$OUTPUT"

stop_glutton
pass "example 3 finished"
