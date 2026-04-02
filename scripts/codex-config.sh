#!/usr/bin/env bash
# codex-config.sh — Backward-compatible wrapper around scripts/bp-config.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/bp-config.sh"

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  set -euo pipefail
  bp_config_main "$@"
fi
