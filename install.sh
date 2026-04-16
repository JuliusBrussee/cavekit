#!/usr/bin/env bash
# Thin compatibility shim around `cavekit install`.

set -euo pipefail

INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

command -v go >/dev/null 2>&1 || {
  echo "go not found." >&2
  exit 1
}

cd "$INSTALL_DIR"
exec go run ./cmd/cavekit install --source-dir "$INSTALL_DIR" "$@"
