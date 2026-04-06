#!/usr/bin/env bash
# codex-gate.sh — T-011: Severity-based gating with fix-task generation
# Source this to get bp_tier_gate function.
#
# Decides whether to block or proceed based on finding severity and config.
# Generates fix tasks for blocking findings.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
if [[ -f "$SCRIPT_DIR/codex-config.sh" ]]; then
  source "$SCRIPT_DIR/codex-config.sh"
else
  bp_config_get() { echo "${2:-}"; }
fi

if [[ -f "$SCRIPT_DIR/codex-findings.sh" ]]; then
  source "$SCRIPT_DIR/codex-findings.sh"
fi

# Guard against double-sourcing
[[ -n "${_BP_GATE_LOADED:-}" ]] && { return 0 2>/dev/null || true; }
_BP_GATE_LOADED=1

# ── bp_tier_gate ───────────────────────────────────────────────────────
# Evaluates findings against tier_gate_mode and returns:
#   0 — proceed (no blocking findings or permissive mode)
#   1 — blocked (has blocking findings that need fix tasks)
#
# Outputs to stdout:
#   GATE_RESULT=proceed|blocked
#   BLOCKING_COUNT=N
#   DEFERRED_COUNT=N
#   BLOCKING_FINDINGS=<newline-separated list of finding IDs>

bp_tier_gate() {
  local mode
  mode="$(bp_config_get tier_gate_mode severity)"

  if [[ "$mode" == "off" ]]; then
    echo "GATE_RESULT=proceed"
    echo "BLOCKING_COUNT=0"
    echo "DEFERRED_COUNT=0"
    return 0
  fi

  local fpath
  fpath="$(bp_findings_path)"

  if [[ ! -f "$fpath" ]]; then
    echo "GATE_RESULT=proceed"
    echo "BLOCKING_COUNT=0"
    echo "DEFERRED_COUNT=0"
    return 0
  fi

  local blocking_ids=""
  local blocking_count=0
  local deferred_count=0

  # Read all NEW findings
  while IFS='|' read -r _ finding severity file status rest; do
    severity="$(echo "$severity" | xargs 2>/dev/null || echo "$severity")"
    status="$(echo "$status" | xargs 2>/dev/null || echo "$status")"
    finding="$(echo "$finding" | xargs 2>/dev/null || echo "$finding")"

    [[ "$status" != "NEW" ]] && continue

    local fid=""
    fid="$(echo "$finding" | grep -oE 'F-[0-9]+' || true)"
    [[ -z "$fid" ]] && continue

    case "$mode" in
      severity)
        # P0/P1 block, P2/P3 deferred
        if [[ "$severity" == "P0" || "$severity" == "P1" ]]; then
          blocking_ids="${blocking_ids}${fid}\n"
          blocking_count=$((blocking_count + 1))
        else
          deferred_count=$((deferred_count + 1))
        fi
        ;;
      strict)
        # All findings block
        blocking_ids="${blocking_ids}${fid}\n"
        blocking_count=$((blocking_count + 1))
        ;;
      permissive)
        # All findings deferred
        deferred_count=$((deferred_count + 1))
        ;;
    esac
  done < <(grep -E '^\|' "$fpath" | grep -vF '| Finding' | grep -vE '^\|[-]')

  echo "GATE_RESULT=$([ $blocking_count -gt 0 ] && echo blocked || echo proceed)"
  echo "BLOCKING_COUNT=$blocking_count"
  echo "DEFERRED_COUNT=$deferred_count"
  if [[ -n "$blocking_ids" ]]; then
    echo "BLOCKING_FINDINGS=$(echo -e "$blocking_ids" | grep -v '^$' | tr '\n' ',' | sed 's/,$//')"
  fi

  [[ $blocking_count -gt 0 ]] && return 1
  return 0
}

# ── bp_generate_fix_tasks ──────────────────────────────────────────────
# Generates fix task descriptions for blocking findings.
# Output: one line per task in format suitable for build loop consumption.
#   FIX-F-NNN|severity|file|description

bp_generate_fix_tasks() {
  local fpath
  fpath="$(bp_findings_path)"

  [[ ! -f "$fpath" ]] && return 0

  local mode
  mode="$(bp_config_get tier_gate_mode severity)"

  while IFS='|' read -r _ finding severity file status rest; do
    severity="$(echo "$severity" | xargs 2>/dev/null || echo "$severity")"
    status="$(echo "$status" | xargs 2>/dev/null || echo "$status")"
    finding="$(echo "$finding" | xargs 2>/dev/null || echo "$finding")"
    file="$(echo "$file" | xargs 2>/dev/null || echo "$file")"

    [[ "$status" != "NEW" ]] && continue

    local fid=""
    fid="$(echo "$finding" | grep -oE 'F-[0-9]+' || true)"
    [[ -z "$fid" ]] && continue

    local is_blocking=false
    case "$mode" in
      severity) [[ "$severity" == "P0" || "$severity" == "P1" ]] && is_blocking=true ;;
      strict) is_blocking=true ;;
    esac

    if [[ "$is_blocking" == "true" ]]; then
      local desc
      desc="$(echo "$finding" | sed "s/^${fid}: //")"
      echo "FIX-${fid}|${severity}|${file}|${desc}"
    fi
  done < <(grep -E '^\|' "$fpath" | grep -vF '| Finding' | grep -vE '^\|[-]')
}

# ── bp_review_fix_cycle ────────────────────────────────────────────────
# Orchestrates the review-fix cycle with max iterations.
# Arguments:
#   $1 — base ref for diff (TIER_START_REF)
#   $2 — max cycles (default: 2)
#
# Returns:
#   0 — all clear (no blocking findings remain)
#   1 — still has blocking findings after max cycles (advance with warning)
#
# This function is called by the build loop after a tier completes.
# It runs codex-review.sh, evaluates the gate, and if blocked, outputs
# the fix tasks for the caller to execute. The caller is responsible for
# actually implementing fixes — this function handles the review/evaluate loop.

bp_review_fix_cycle() {
  local base_ref="${1:?base ref required}"
  local max_cycles="${2:-2}"
  local cycle=0

  while (( cycle < max_cycles )); do
    cycle=$((cycle + 1))
    echo "[ck:tier-gate] Review-fix cycle ${cycle}/${max_cycles}"

    # Run the review
    if [[ -f "$SCRIPT_DIR/codex-review.sh" ]]; then
      bash "$SCRIPT_DIR/codex-review.sh" --base "$base_ref"
    else
      echo "[ck:tier-gate] codex-review.sh not found, skipping review"
      return 0
    fi

    # Evaluate the gate
    local gate_output
    gate_output="$(bp_tier_gate)" || true

    local gate_result blocking_count
    gate_result="$(echo "$gate_output" | grep 'GATE_RESULT=' | cut -d= -f2)"
    blocking_count="$(echo "$gate_output" | grep 'BLOCKING_COUNT=' | cut -d= -f2)"

    echo "$gate_output"

    if [[ "$gate_result" == "proceed" ]]; then
      echo "[ck:tier-gate] Gate: PROCEED (no blocking findings)"
      return 0
    fi

    # If blocked and not the last cycle, output fix tasks for the caller
    if (( cycle < max_cycles )); then
      echo "[ck:tier-gate] Gate: BLOCKED — ${blocking_count} finding(s) need fixes"
      echo "[ck:tier-gate] Fix tasks for this cycle:"
      bp_generate_fix_tasks
      echo "[ck:tier-gate] AWAITING_FIXES"
      # The caller implements fixes, marks findings FIXED, then calls us again
      # by continuing the loop. In practice, the build loop reads AWAITING_FIXES,
      # implements the fixes, and then the loop continues with the re-review.
      return 2  # Signal: fixes needed, caller should implement and re-call
    fi
  done

  # Exhausted max cycles — advance with warning
  local remaining
  remaining="$(bp_generate_fix_tasks | wc -l | tr -d ' ')"
  echo "[ck:tier-gate] WARNING: Advancing after ${max_cycles} review-fix cycles with ${remaining} unresolved blocking findings"
  return 1
}

# ── CLI mode ───────────────────────────────────────────────────────────

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  set -euo pipefail
  cmd="${1:-gate}"
  shift || true
  case "$cmd" in
    gate) bp_tier_gate ;;
    fix-tasks) bp_generate_fix_tasks ;;
    cycle) bp_review_fix_cycle "$@" ;;
    help|--help|-h)
      echo "Usage: codex-gate.sh {gate|fix-tasks|cycle}"
      echo "  gate              Evaluate findings and decide block/proceed"
      echo "  fix-tasks         Generate fix task list for blocking findings"
      echo "  cycle <ref> [max] Run review-fix cycle (default max: 2)"
      ;;
    *) echo "Unknown: $cmd" >&2; exit 1 ;;
  esac
fi
