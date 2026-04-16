#!/usr/bin/env bash
# Thin compatibility shim around `cavekit sync-codex`.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

exec "$SCRIPT_DIR/cavekit" sync-codex --source-dir "$ROOT_DIR" "$@"
