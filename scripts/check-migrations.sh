#!/usr/bin/env bash
# Verifies that every SQL migration file contains an explicit BEGIN TRANSACTION
# and COMMIT, which is required because the golang-migrate driver runs with
# NoTxWrap: true (no implicit transaction wrapping).
set -euo pipefail

MIGRATIONS_DIR="${1:-$(dirname "$0")/../internal/database/migrations}"
MIGRATIONS_DIR="$(cd "$MIGRATIONS_DIR" && pwd)"

fail=0

for f in "$MIGRATIONS_DIR"/*.sql; do
  if ! grep -q "^BEGIN" "$f"; then
    echo "❌ Missing BEGIN TRANSACTION: ${f##*/}"
    fail=1
  fi
  if ! grep -q "^COMMIT" "$f"; then
    echo "❌ Missing COMMIT: ${f##*/}"
    fail=1
  fi
done

if [ "$fail" -eq 0 ]; then
  echo "✅ All migration files have explicit BEGIN TRANSACTION / COMMIT."
fi

exit "$fail"
