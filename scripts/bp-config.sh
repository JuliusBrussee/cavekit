#!/usr/bin/env bash
# bp-config.sh — Blueprint configuration utilities
#
# Canonical config helper for Blueprint settings.
# Supports user-level defaults (~/.blueprint/config) and project-level overrides
# (.blueprint/config under the repo root). Project values always win.
#
# Provides:
#   bp_config_get <key> [default]
#   bp_config_set <key> <value> [--global|--project]
#   bp_config_init
#   bp_config_model <reasoning|execution|exploration>
#   bp_config_effective_preset
#   bp_config_summary_line

if [[ -n "${_BP_CONFIG_LOADED:-}" ]]; then
  return 0 2>/dev/null || true
fi
_BP_CONFIG_LOADED=1

_bp_config_default() {
  case "$1" in
    bp_model_preset) echo "quality" ;;
    codex_review) echo "auto" ;;
    codex_model) echo "" ;;
    codex_effort) echo "" ;;
    tier_gate_mode) echo "severity" ;;
    command_gate) echo "all" ;;
    command_gate_model) echo "o4-mini" ;;
    command_gate_timeout) echo "3000" ;;
    command_gate_allowlist) echo "" ;;
    command_gate_blocklist) echo "" ;;
    speculative_review) echo "" ;;
    speculative_review_timeout) echo "" ;;
    *) echo "" ;;
  esac
}

_bp_config_validate() {
  local key="$1" value="$2"

  case "$key" in
    bp_model_preset)
      case "$value" in expensive|quality|balanced|fast) return 0 ;; esac
      echo "bp_config_set: invalid value '$value' for '$key' (allowed: expensive quality balanced fast)" >&2
      return 1
      ;;
    codex_review)
      case "$value" in auto|off) return 0 ;; esac
      echo "bp_config_set: invalid value '$value' for '$key' (allowed: auto off)" >&2
      return 1
      ;;
    tier_gate_mode)
      case "$value" in severity|strict|permissive|off) return 0 ;; esac
      echo "bp_config_set: invalid value '$value' for '$key' (allowed: severity strict permissive off)" >&2
      return 1
      ;;
    command_gate)
      case "$value" in all|interactive|off) return 0 ;; esac
      echo "bp_config_set: invalid value '$value' for '$key' (allowed: all interactive off)" >&2
      return 1
      ;;
    command_gate_timeout|speculative_review_timeout)
      case "$value" in ''|*[!0-9]*) ;;
        *) return 0 ;;
      esac
      echo "bp_config_set: invalid value '$value' for '$key' (allowed: positive integer)" >&2
      return 1
      ;;
    speculative_review)
      case "$value" in on|off) return 0 ;; esac
      echo "bp_config_set: invalid value '$value' for '$key' (allowed: on off)" >&2
      return 1
      ;;
    *) return 0 ;;
  esac
}

_BP_CONFIG_KEYS="bp_model_preset codex_review codex_model codex_effort tier_gate_mode command_gate command_gate_model command_gate_timeout command_gate_allowlist command_gate_blocklist speculative_review speculative_review_timeout"

bp_global_config_path() {
  if [[ -n "${BP_GLOBAL_CONFIG_PATH:-}" ]]; then
    echo "$BP_GLOBAL_CONFIG_PATH"
    return
  fi
  echo "${HOME}/.blueprint/config"
}

bp_project_config_path() {
  if [[ -n "${BP_PROJECT_CONFIG_PATH:-}" ]]; then
    echo "$BP_PROJECT_CONFIG_PATH"
    return
  fi

  local root="${BP_PROJECT_ROOT:-}"
  if [[ -z "$root" ]]; then
    root="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")"
  fi
  echo "${root}/.blueprint/config"
}

bp_config_path() {
  bp_project_config_path
}

_bp_config_read_file_value() {
  local key="$1" file="$2"

  if [[ -f "$file" ]]; then
    grep -E "^${key}=" "$file" 2>/dev/null | tail -1 | cut -d'=' -f2- || true
  fi
}

bp_config_get_source() {
  local key="${1:?bp_config_get_source: key required}"
  local project_cfg global_cfg project_val global_val

  project_cfg="$(bp_project_config_path)"
  global_cfg="$(bp_global_config_path)"
  project_val="$(_bp_config_read_file_value "$key" "$project_cfg")"
  global_val="$(_bp_config_read_file_value "$key" "$global_cfg")"

  if [[ -n "$project_val" ]]; then
    echo "project"
  elif [[ -n "$global_val" ]]; then
    echo "global"
  else
    echo "default"
  fi
}

bp_config_get_source_path() {
  local key="${1:?bp_config_get_source_path: key required}"
  local source

  source="$(bp_config_get_source "$key")"
  case "$source" in
    project) bp_project_config_path ;;
    global) bp_global_config_path ;;
    *) echo "(built-in default)" ;;
  esac
}

bp_config_get() {
  local key="${1:?bp_config_get: key required}"
  local fallback="${2:-$(_bp_config_default "$key")}"
  local project_cfg global_cfg project_val global_val

  project_cfg="$(bp_project_config_path)"
  global_cfg="$(bp_global_config_path)"
  project_val="$(_bp_config_read_file_value "$key" "$project_cfg")"
  global_val="$(_bp_config_read_file_value "$key" "$global_cfg")"

  if [[ -n "$project_val" ]]; then
    echo "$project_val"
    return 0
  fi

  if [[ -n "$global_val" ]]; then
    echo "$global_val"
    return 0
  fi

  echo "$fallback"
}

_bp_config_parse_set_args() {
  BP_CONFIG_SET_SCOPE="project"
  BP_CONFIG_SET_KEY=""
  BP_CONFIG_SET_VALUE=""
  BP_CONFIG_SET_VALUE_SEEN="0"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --global)
        BP_CONFIG_SET_SCOPE="global"
        shift
        ;;
      --project)
        BP_CONFIG_SET_SCOPE="project"
        shift
        ;;
      *)
        if [[ -z "$BP_CONFIG_SET_KEY" ]]; then
          BP_CONFIG_SET_KEY="$1"
        elif [[ "$BP_CONFIG_SET_VALUE_SEEN" == "0" ]]; then
          BP_CONFIG_SET_VALUE="$1"
          BP_CONFIG_SET_VALUE_SEEN="1"
        else
          echo "bp_config_set: unexpected argument '$1'" >&2
          return 1
        fi
        shift
        ;;
    esac
  done

  if [[ -z "$BP_CONFIG_SET_KEY" ]]; then
    echo "bp_config_set: key required" >&2
    return 1
  fi

  if [[ "$BP_CONFIG_SET_VALUE_SEEN" != "1" ]]; then
    echo "bp_config_set: value required" >&2
    return 1
  fi

  return 0
}

bp_config_set() {
  _bp_config_parse_set_args "$@" || return 1

  local key="$BP_CONFIG_SET_KEY"
  local value="$BP_CONFIG_SET_VALUE"
  local cfg

  _bp_config_validate "$key" "$value" || return 1

  if [[ "$BP_CONFIG_SET_SCOPE" == "global" ]]; then
    cfg="$(bp_global_config_path)"
  else
    cfg="$(bp_project_config_path)"
  fi

  mkdir -p "$(dirname "$cfg")"

  if [[ -f "$cfg" ]] && grep -qE "^${key}=" "$cfg" 2>/dev/null; then
    local tmp="${cfg}.tmp.$$"
    sed "s|^${key}=.*|${key}=${value}|" "$cfg" > "$tmp" && mv "$tmp" "$cfg"
  else
    echo "${key}=${value}" >> "$cfg"
  fi
}

_bp_config_init_global_file() {
  local cfg="$1"
  local key

  mkdir -p "$(dirname "$cfg")"

  if [[ ! -f "$cfg" ]]; then
    {
      echo "# Blueprint configuration"
      echo "# User-level defaults"
      echo "# See: scripts/bp-config.sh for documentation"
      echo
      for key in $_BP_CONFIG_KEYS; do
        echo "${key}=$(_bp_config_default "$key")"
      done
    } > "$cfg"
    return 0
  fi

  for key in $_BP_CONFIG_KEYS; do
    if ! grep -qE "^${key}=" "$cfg" 2>/dev/null; then
      echo "${key}=$(_bp_config_default "$key")" >> "$cfg"
    fi
  done
}

_bp_config_init_project_file() {
  local cfg="$1"

  mkdir -p "$(dirname "$cfg")"

  if [[ ! -f "$cfg" ]]; then
    cat > "$cfg" <<'EOF'
# Blueprint configuration
# Project-level overrides
# Add only the keys you want this repo to override.
# See: scripts/bp-config.sh for documentation
EOF
  fi
}

bp_config_init() {
  _bp_config_init_global_file "$(bp_global_config_path)"
  _bp_config_init_project_file "$(bp_project_config_path)"
}

bp_config_effective_preset() {
  local preset

  preset="$(bp_config_get bp_model_preset quality)"
  case "$preset" in
    expensive|quality|balanced|fast)
      echo "$preset"
      ;;
    *)
      echo "bp_config_effective_preset: invalid preset '$preset' (allowed: expensive quality balanced fast)" >&2
      return 1
      ;;
  esac
}

bp_config_model() {
  local task_type="${1:?bp_config_model: task type required}"
  local preset

  preset="$(bp_config_effective_preset)" || return 1

  case "$preset:$task_type" in
    expensive:reasoning|expensive:execution|expensive:exploration)
      echo "opus"
      ;;
    quality:reasoning|quality:execution)
      echo "opus"
      ;;
    quality:exploration)
      echo "sonnet"
      ;;
    balanced:reasoning)
      echo "opus"
      ;;
    balanced:execution)
      echo "sonnet"
      ;;
    balanced:exploration|fast:exploration)
      echo "haiku"
      ;;
    fast:reasoning|fast:execution)
      echo "sonnet"
      ;;
    *)
      echo "bp_config_model: unknown task type '$task_type' (allowed: reasoning execution exploration)" >&2
      return 1
      ;;
  esac
}

bp_config_summary_line() {
  local preset reasoning execution exploration

  preset="$(bp_config_effective_preset)" || return 1
  reasoning="$(bp_config_model reasoning)" || return 1
  execution="$(bp_config_model execution)" || return 1
  exploration="$(bp_config_model exploration)" || return 1

  echo "Blueprint preset: ${preset} (reasoning=${reasoning}, execution=${execution}, exploration=${exploration})"
}

bp_config_show() {
  local preset preset_source preset_source_path
  local reasoning execution exploration

  preset="$(bp_config_effective_preset)" || return 1
  preset_source="$(bp_config_get_source bp_model_preset)"
  preset_source_path="$(bp_config_get_source_path bp_model_preset)"
  reasoning="$(bp_config_model reasoning)" || return 1
  execution="$(bp_config_model execution)" || return 1
  exploration="$(bp_config_model exploration)" || return 1

  cat <<EOF
bp_model_preset=${preset}
bp_model_preset_source=${preset_source}
bp_model_preset_source_path=${preset_source_path}
reasoning_model=${reasoning}
execution_model=${execution}
exploration_model=${exploration}
project_config=$(bp_project_config_path)
global_config=$(bp_global_config_path)
EOF
}

bp_config_presets() {
  cat <<'EOF'
| Preset | Reasoning | Execution | Exploration |
|---|---|---|---|
| expensive | opus | opus | opus |
| quality | opus | opus | sonnet |
| balanced | opus | sonnet | haiku |
| fast | sonnet | sonnet | haiku |
EOF
}

bp_config_list() {
  local scope="project"
  local cfg=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --global) scope="global"; shift ;;
      --project) scope="project"; shift ;;
      *) echo "bp_config_list: unknown argument '$1'" >&2; return 1 ;;
    esac
  done

  if [[ "$scope" == "global" ]]; then
    cfg="$(bp_global_config_path)"
  else
    cfg="$(bp_project_config_path)"
  fi

  if [[ -f "$cfg" ]]; then
    grep -E '^[a-z_]+=.*' "$cfg" 2>/dev/null || true
  fi
}

bp_config_main() {
  local cmd="${1:-help}"
  shift || true

  case "$cmd" in
    init) bp_config_init ;;
    get) bp_config_get "$@" ;;
    set) bp_config_set "$@" ;;
    list) bp_config_list "$@" ;;
    path)
      case "${1:-}" in
        --global) bp_global_config_path ;;
        --project|"") bp_project_config_path ;;
        *) echo "bp-config.sh path: unknown argument '${1}'" >&2; return 1 ;;
      esac
      ;;
    source) bp_config_get_source "$@" ;;
    source-path) bp_config_get_source_path "$@" ;;
    effective-preset) bp_config_effective_preset ;;
    model) bp_config_model "$@" ;;
    show) bp_config_show ;;
    summary) bp_config_summary_line ;;
    presets) bp_config_presets ;;
    help|--help|-h)
      cat <<'EOF'
Usage: bp-config.sh {init|get|set|list|path|source|source-path|effective-preset|model|show|summary|presets}
  init                              Create/backfill global and project config files
  get <key> [default]               Read an effective config value
  set <key> <val> [--global|--project]
                                    Write a config value (project by default)
  list [--global|--project]         Show raw config key=value pairs from one file
  path [--global|--project]         Print a config file path
  source <key>                      Print value source: project | global | default
  source-path <key>                 Print the path that supplied the value
  effective-preset                  Print the effective model preset
  model <task-type>                 Resolve model for reasoning | execution | exploration
  show                              Print effective preset, source, and resolved models
  summary                           Print a one-line preset summary
  presets                           Print the built-in preset table
EOF
      ;;
    *)
      echo "Unknown command: $cmd" >&2
      return 1
      ;;
  esac
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  set -euo pipefail
  bp_config_main "$@"
fi
