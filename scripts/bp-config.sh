#!/usr/bin/env bash
# Thin compatibility shim around `cavekit config`.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

bp_config_init() { "$SCRIPT_DIR/cavekit" config init "$@"; }
bp_config_get() { "$SCRIPT_DIR/cavekit" config get "$@"; }
bp_config_set() { "$SCRIPT_DIR/cavekit" config set "$@"; }
bp_config_list() { "$SCRIPT_DIR/cavekit" config list "$@"; }
bp_global_config_path() { "$SCRIPT_DIR/cavekit" config path --global "$@"; }
bp_project_config_path() { "$SCRIPT_DIR/cavekit" config path --project "$@"; }
bp_config_path() { "$SCRIPT_DIR/cavekit" config path --project "$@"; }
bp_config_get_source() { "$SCRIPT_DIR/cavekit" config source "$@"; }
bp_config_get_source_path() { "$SCRIPT_DIR/cavekit" config source-path "$@"; }
bp_config_effective_preset() { "$SCRIPT_DIR/cavekit" config effective-preset "$@"; }
bp_config_model() { "$SCRIPT_DIR/cavekit" config model "$@"; }
bp_config_show() { "$SCRIPT_DIR/cavekit" config show "$@"; }
bp_config_summary_line() { "$SCRIPT_DIR/cavekit" config summary "$@"; }
bp_config_presets() { "$SCRIPT_DIR/cavekit" config presets "$@"; }
bp_config_caveman_active() { "$SCRIPT_DIR/cavekit" config caveman-active "$@"; }

bp_config_main() {
  "$SCRIPT_DIR/cavekit" config "$@"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  set -euo pipefail
  bp_config_main "$@"
fi
