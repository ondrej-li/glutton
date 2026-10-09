#!/usr/bin/env bash
# Example 4: HTTPS using your own certificate.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

OUTPUT=/tmp/glutton-examples/tls-certificate
CERT_DIR="$WORK_DIR/certs"
URL=https://127.0.0.1:4357/v1/glutton/save

command -v openssl >/dev/null 2>&1 || {
  skip "openssl is required to generate the example certificate"
  exit 0
}

build_glutton
rm -rf "$OUTPUT"
mkdir -p "$CERT_DIR"

log "generating a certificate with openssl"
openssl req -x509 -newkey rsa:2048 \
  -keyout "$CERT_DIR/key.pem" -out "$CERT_DIR/cert.pem" \
  -days 1 -nodes -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" >/dev/null 2>&1 ||
  fail "could not generate a certificate, OpenSSL 1.1.1 or newer is required for -addext"

# the certificate and key paths are not in the config file, they are passed
# through the environment, which takes precedence over the yaml file
export CERT_FILE="$CERT_DIR/cert.pem"
export KEY_FILE="$CERT_DIR/key.pem"

start_glutton "$REPO_ROOT/examples/configs/tls-certificate.yaml"
wait_for_glutton "$URL" --cacert "$CERT_DIR/cert.pem"

log "without the CA the certificate is not trusted"
if curl -s -o /dev/null "$URL" 2>/dev/null; then
  fail "expected certificate verification to fail"
fi
pass "certificate verification fails without --cacert"

log "posting a payload with the certificate verified"
expect_status 200 "$URL" --cacert "$CERT_DIR/cert.pem" -X POST -d '{"hello":"glutton"}'

log "payloads stored in $OUTPUT:"
ls -1 "$OUTPUT"

stop_glutton
pass "example 4 finished"
