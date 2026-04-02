#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_SCRIPT="$ROOT_DIR/scripts/bp-config.sh"
LEGACY_SCRIPT="$ROOT_DIR/scripts/codex-config.sh"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

PROJECT_ROOT="$TMPDIR/project"
GLOBAL_CONFIG="$TMPDIR/home/.blueprint/config"

mkdir -p "$PROJECT_ROOT"

run_cfg() {
  BP_PROJECT_ROOT="$PROJECT_ROOT" \
  BP_GLOBAL_CONFIG_PATH="$GLOBAL_CONFIG" \
  bash "$CONFIG_SCRIPT" "$@"
}

run_legacy_cfg() {
  BP_PROJECT_ROOT="$PROJECT_ROOT" \
  BP_GLOBAL_CONFIG_PATH="$GLOBAL_CONFIG" \
  bash "$LEGACY_SCRIPT" "$@"
}

assert_eq() {
  local expected="$1"
  local actual="$2"
  local label="$3"

  if [[ "$expected" != "$actual" ]]; then
    echo "FAIL: $label" >&2
    echo "  expected: $expected" >&2
    echo "  actual:   $actual" >&2
    exit 1
  fi
}

assert_fail() {
  local label="$1"
  shift

  if "$@" >/dev/null 2>&1; then
    echo "FAIL: $label" >&2
    echo "  expected command to fail" >&2
    exit 1
  fi
}

assert_eq "quality" "$(run_cfg effective-preset)" "default preset"
assert_eq "opus" "$(run_cfg model reasoning)" "default reasoning model"
assert_eq "opus" "$(run_cfg model execution)" "default execution model"
assert_eq "sonnet" "$(run_cfg model exploration)" "default exploration model"
assert_eq "default" "$(run_cfg source bp_model_preset)" "default source"

run_cfg init
run_cfg set bp_model_preset fast --global
assert_eq "fast" "$(run_cfg effective-preset)" "global preset"
assert_eq "global" "$(run_cfg source bp_model_preset)" "global source"
assert_eq "sonnet" "$(run_cfg model reasoning)" "fast reasoning model"
assert_eq "haiku" "$(run_cfg model exploration)" "fast exploration model"

run_cfg set bp_model_preset balanced --project
assert_eq "balanced" "$(run_cfg effective-preset)" "project override preset"
assert_eq "project" "$(run_cfg source bp_model_preset)" "project source"
assert_eq "opus" "$(run_cfg model reasoning)" "balanced reasoning model"
assert_eq "sonnet" "$(run_cfg model execution)" "balanced execution model"
assert_eq "haiku" "$(run_cfg model exploration)" "balanced exploration model"

assert_fail "invalid preset rejected" run_cfg set bp_model_preset invalid --project

run_cfg set codex_model gpt-5.4-mini --project
assert_eq "gpt-5.4-mini" "$(run_cfg get codex_model)" "existing codex setting round-trip"

assert_eq "gpt-5.4-mini" "$(run_cfg get codex_model)" "init preserves existing project value"
assert_eq "balanced" "$(run_legacy_cfg get bp_model_preset)" "legacy wrapper compatibility"

echo "bp-config tests passed"
