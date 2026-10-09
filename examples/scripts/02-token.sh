#!/usr/bin/env bash
# Example 2: a token protected route.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

OUTPUT=/tmp/glutton-examples/token
BASE=http://127.0.0.1:4355/v1/glutton/save
KEY=0123456789abcdef

build_glutton
rm -rf "$OUTPUT"
start_glutton "$REPO_ROOT/examples/configs/token.yaml"
wait_for_glutton "$BASE"

log "asking for a token without the key is rejected"
expect_status 412 "$BASE/token"

log "asking for a token with the key"
TOKEN="$(curl -s -H "token-key: $KEY" "$BASE/token")"
[ -n "$TOKEN" ] || fail "the token endpoint returned an empty token"
pass "got a ${#TOKEN} byte token"

log "saving without a token is rejected"
expect_status 412 "$BASE" -X POST -d '{"hello":"glutton"}'

log "saving with the token"
expect_status 200 "$BASE" -X POST -H "token: $TOKEN" -d '{"hello":"glutton"}'

log "payloads stored in $OUTPUT:"
ls -1 "$OUTPUT"

stop_glutton
pass "example 2 finished"
