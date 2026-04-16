#!/usr/bin/env bash
# Thin compatibility shim around `cavekit codex-review`.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

bp_codex_review() {
  "$SCRIPT_DIR/cavekit" codex-review "$@"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  set -euo pipefail
  bp_codex_review "$@"
fi
