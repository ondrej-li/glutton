#!/usr/bin/env bash
# Example 5: storing payloads in Postgres.
#
# Point it at another database with:
#   GLUTTON_POSTGRES_DSN='postgres://user:pass@host:5432/db?sslmode=disable' bash 05-postgres.sh
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

DSN="${GLUTTON_POSTGRES_DSN:-postgres://postgres:postgres@127.0.0.1:5432/glutton?sslmode=disable}"
URL=http://127.0.0.1:4358/v1/glutton/save
CONFIG="$WORK_DIR/postgres.yaml"

command -v psql >/dev/null 2>&1 || {
  skip "psql is required to prepare the schema and verify the result"
  exit 0
}
# -w makes psql fail instead of prompting for a password
psql -w "$DSN" -c 'select 1' >/dev/null 2>&1 || {
  skip "cannot reach $DSN, set GLUTTON_POSTGRES_DSN to a reachable database"
  exit 0
}

build_glutton
mkdir -p "$WORK_DIR"

# the connection string lives in the route settings, and environment variables
# only configure a route when the yaml file defines none - hence this config
cat >"$CONFIG" <<EOF
host: 127.0.0.1
port: "4358"
settings:
  - name: postgres example
    uri: save
    parser: SimpleParser
    notifier: NilNotifier
    saver: DatabaseSaver
    sql_driver: postgres
    sql_connection_string: "$DSN"
EOF

log "applying examples/sql/schema.sql"
psql -w "$DSN" -f "$REPO_ROOT/examples/sql/schema.sql" >/dev/null

MARK="glutton-example-$(date +%s)"
before="$(psql -w "$DSN" -tAc "select count(*) from payload where payload like '%$MARK%'")"

start_glutton "$CONFIG"
wait_for_glutton "$URL"

log "posting a payload"
expect_status 200 "$URL" -X POST -H 'Content-Type: application/json' -d "{\"mark\":\"$MARK\"}"

after="$(psql -w "$DSN" -tAc "select count(*) from payload where payload like '%$MARK%'")"
[ "$after" -gt "$before" ] || fail "no row was stored for $MARK"
pass "found the stored row for $MARK"

log "the stored row:"
psql -w "$DSN" -c "select ts, remote, meta, payload from payload where payload like '%$MARK%' order by ts desc limit 1"

log "rows in the table now: $(psql -w "$DSN" -tAc 'select count(*) from payload')"

stop_glutton
pass "example 5 finished"
