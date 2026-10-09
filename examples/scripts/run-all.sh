#!/usr/bin/env bash
# Run every example in order. Examples that need extra tooling skip themselves.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

for script in "$here"/0*.sh; do
  printf '\n==============================================================\n'
  printf 'running %s\n' "$(basename "$script")"
  printf '==============================================================\n'
  bash "$script"
done

printf '\nall examples finished\n'
