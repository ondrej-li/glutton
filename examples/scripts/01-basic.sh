#!/usr/bin/env bash
# Example 1: a single route that stores payloads on the filesystem.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

OUTPUT=/tmp/glutton-examples/basic
URL=http://127.0.0.1:4354/v1/glutton/save

build_glutton
rm -rf "$OUTPUT"
start_glutton "$REPO_ROOT/examples/configs/basic.yaml"
wait_for_glutton "$URL"

log "posting a payload"
expect_status 200 "$URL" -X POST -H 'Content-Type: application/json' -d '{"hello":"glutton"}'

log "payloads stored in $OUTPUT:"
ls -1 "$OUTPUT"

log "the saved file contains the payload, the remote address and the meta data:"
cat "$OUTPUT"/*

stop_glutton
pass "example 1 finished"
