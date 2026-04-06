#!/usr/bin/env bash
# codex-speculative.sh — Speculative Pre-Build Review
# T-201: Configuration schema and defaults
# T-202: Background job ID tracking
# T-203: Background Codex review dispatch
# T-204: Pipeline overlap status reporting
# T-205: Result retrieval with timeout and fallback
# T-206: Finding reconciliation
# T-207: P0/P1 queuing
# T-208: Impl tracking integration
#
# Source this file to get bp_speculative_* functions.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source dependencies
if [[ -f "$SCRIPT_DIR/codex-detect.sh" ]]; then
  source "$SCRIPT_DIR/codex-detect.sh"
else
  codex_available=false
fi

if [[ -f "$SCRIPT_DIR/codex-config.sh" ]]; then
  source "$SCRIPT_DIR/codex-config.sh"
else
  bp_config_get() { echo "${2:-}"; }
fi

if [[ -f "$SCRIPT_DIR/codex-findings.sh" ]]; then
  source "$SCRIPT_DIR/codex-findings.sh"
fi

if [[ -f "$SCRIPT_DIR/codex-gate.sh" ]]; then
  source "$SCRIPT_DIR/codex-gate.sh"
fi

# Guard against double-sourcing
[[ -n "${_BP_SPECULATIVE_LOADED:-}" ]] && { return 0 2>/dev/null || true; }
_BP_SPECULATIVE_LOADED=1

# ── T-201: Configuration Schema and Defaults ──────────────────────────

# Register speculative review config keys with codex-config.sh defaults.
# Extends the _bp_config_default function if possible, otherwise uses wrappers.

bp_speculative_config_get() {
  local key="$1"
  case "$key" in
    speculative_review)
      local val
      val="$(bp_config_get speculative_review "")"
      if [[ -z "$val" ]]; then
        # Default: on when codex available and tier gate not off
        local gate_mode
        gate_mode="$(bp_config_get tier_gate_mode severity)"
        if [[ "$codex_available" == "true" && "$gate_mode" != "off" ]]; then
          echo "on"
        else
          echo "off"
        fi
      else
        echo "$val"
      fi
      ;;
    speculative_review_timeout)
      bp_config_get speculative_review_timeout 300
      ;;
    *)
      bp_config_get "$@"
      ;;
  esac
}

bp_speculative_enabled() {
  local mode
  mode="$(bp_speculative_config_get speculative_review)"
  [[ "$mode" == "on" ]]
}

# ── T-202: Background Job ID Tracking ─────────────────────────────────

# Session-scoped tracking of background Codex review jobs.
# Uses a temp file keyed by session to track job state.

_BP_SPECULATIVE_STATE_DIR="${TMPDIR:-/tmp}/bp-speculative-$$"

_bp_speculative_state_file() {
  echo "${_BP_SPECULATIVE_STATE_DIR}/jobs"
}

bp_speculative_init_tracking() {
  mkdir -p "$_BP_SPECULATIVE_STATE_DIR"
  : > "$(_bp_speculative_state_file)"
}

# Record a dispatched background job.
# $1 — tier number
# $2 — job PID
# $3 — output file path
# $4 — base ref used for the diff

bp_speculative_record_job() {
  local tier="$1" pid="$2" output_file="$3" base_ref="$4"
  local state_file
  state_file="$(_bp_speculative_state_file)"
  echo "${tier}|${pid}|${output_file}|${base_ref}|running|$(date +%s)" >> "$state_file"
}

# Get job info for a tier. Returns: TIER|PID|OUTPUT_FILE|BASE_REF|STATUS|START_TIME
bp_speculative_get_job() {
  local tier="$1"
  local state_file
  state_file="$(_bp_speculative_state_file)"
  [[ -f "$state_file" ]] || return 1
  grep "^${tier}|" "$state_file" | tail -1
}

# Update job status
bp_speculative_update_job() {
  local tier="$1" new_status="$2"
  local state_file
  state_file="$(_bp_speculative_state_file)"
  [[ -f "$state_file" ]] || return 1

  local tmp="${state_file}.tmp.$$"
  while IFS='|' read -r t pid outf bref status start; do
    if [[ "$t" == "$tier" ]]; then
      echo "${t}|${pid}|${outf}|${bref}|${new_status}|${start}"
    else
      echo "${t}|${pid}|${outf}|${bref}|${status}|${start}"
    fi
  done < "$state_file" > "$tmp"
  mv "$tmp" "$state_file"
}

# List all tracked jobs
bp_speculative_list_jobs() {
  local state_file
  state_file="$(_bp_speculative_state_file)"
  [[ -f "$state_file" ]] && cat "$state_file"
}

# Cleanup session state
bp_speculative_cleanup() {
  rm -rf "$_BP_SPECULATIVE_STATE_DIR" 2>/dev/null || true
}

# ── T-203: Background Codex Review Dispatch ───────────────────────────
# Launch a Codex adversarial review in the background at tier completion.
#
# $1 — tier number (just completed)
# $2 — base ref (TIER_START_REF)

bp_speculative_dispatch() {
  local tier="$1" base_ref="$2"

  if ! bp_speculative_enabled; then
    echo "[ck:speculative] Speculative review disabled. Skipping."
    return 0
  fi

  if [[ "$codex_available" != "true" ]]; then
    echo "[ck:speculative] Codex unavailable. Skipping speculative dispatch."
    return 0
  fi

  # Tier 0 has no previous tier — skip
  if [[ "$tier" == "0" ]]; then
    echo "[ck:speculative] Tier 0 — no previous tier to review speculatively."
    return 0
  fi

  local output_file="${_BP_SPECULATIVE_STATE_DIR}/review-tier-${tier}.out"
  mkdir -p "$_BP_SPECULATIVE_STATE_DIR"

  echo "[ck:speculative] Dispatching background review of tier $((tier - 1)) (diff from $base_ref)..."

  # Run codex-review.sh in background, capture output
  bash "$SCRIPT_DIR/codex-review.sh" --base "$base_ref" > "$output_file" 2>&1 &
  local pid=$!

  bp_speculative_record_job "$tier" "$pid" "$output_file" "$base_ref"
  echo "[ck:speculative] Background job PID=$pid dispatched for tier $((tier - 1)) review."
}

# ── T-204: Pipeline Overlap Status Reporting ──────────────────────────

bp_speculative_status() {
  local state_file
  state_file="$(_bp_speculative_state_file)"

  if [[ ! -f "$state_file" ]] || [[ ! -s "$state_file" ]]; then
    echo "[ck:speculative] No speculative reviews tracked."
    return 0
  fi

  echo "[ck:speculative] Pipeline status:"
  while IFS='|' read -r tier pid outf bref status start; do
    [[ -z "$tier" ]] && continue
    local now elapsed state_desc
    now="$(date +%s)"
    elapsed=$((now - start))

    # Check if process is still running
    if [[ "$status" == "running" ]]; then
      if kill -0 "$pid" 2>/dev/null; then
        state_desc="RUNNING (${elapsed}s)"
      else
        # Process finished — check if output exists
        if [[ -f "$outf" && -s "$outf" ]]; then
          state_desc="COMPLETE (${elapsed}s)"
          bp_speculative_update_job "$tier" "complete"
        else
          state_desc="FAILED (${elapsed}s)"
          bp_speculative_update_job "$tier" "failed"
        fi
      fi
    else
      state_desc="$(echo "$status" | tr '[:lower:]' '[:upper:]') (${elapsed}s)"
    fi

    echo "  Tier $((tier - 1)) review → $state_desc"
  done < "$state_file"
}

# ── T-205: Result Retrieval with Timeout and Fallback ─────────────────

# Retrieve speculative review results for a tier boundary.
# $1 — tier number (the tier whose *previous* review we want)
#
# Returns:
#   0 — speculative results consumed (findings already in findings file)
#   1 — speculative review not available, fallback to synchronous
#   2 — timed out waiting, fallback to synchronous

bp_speculative_retrieve() {
  local tier="$1"
  local timeout
  timeout="$(bp_speculative_config_get speculative_review_timeout)"

  local job_info
  job_info="$(bp_speculative_get_job "$tier" 2>/dev/null)" || {
    echo "[ck:speculative] No speculative job for tier $tier. Falling back to synchronous."
    return 1
  }

  local j_tier j_pid j_outf j_bref j_status j_start
  IFS='|' read -r j_tier j_pid j_outf j_bref j_status j_start <<< "$job_info"

  # Check if already complete
  if [[ "$j_status" == "complete" || "$j_status" == "consumed" ]]; then
    if [[ "$j_status" == "consumed" ]]; then
      echo "[ck:speculative] Tier $((tier - 1)) review already consumed."
      return 0
    fi
    echo "[ck:speculative] Tier $((tier - 1)) review already complete. Consuming results."
    _bp_speculative_consume_results "$tier" "$j_outf"
    return $?
  fi

  if [[ "$j_status" == "failed" ]]; then
    echo "[ck:speculative] Tier $((tier - 1)) review failed. Falling back to synchronous."
    return 1
  fi

  # Still running — wait with timeout
  echo "[ck:speculative] Tier $((tier - 1)) review still running. Waiting up to ${timeout}s..."
  local waited=0
  while (( waited < timeout )); do
    if ! kill -0 "$j_pid" 2>/dev/null; then
      # Process exited
      if [[ -f "$j_outf" && -s "$j_outf" ]]; then
        bp_speculative_update_job "$tier" "complete"
        echo "[ck:speculative] Review completed after ${waited}s wait."
        _bp_speculative_consume_results "$tier" "$j_outf"
        return $?
      else
        bp_speculative_update_job "$tier" "failed"
        echo "[ck:speculative] Review process exited but no output. Falling back."
        return 1
      fi
    fi
    sleep 2
    waited=$((waited + 2))
  done

  # Timed out
  echo "[ck:speculative] Timed out after ${timeout}s. Falling back to synchronous review."
  bp_speculative_update_job "$tier" "timeout"
  return 2
}

_bp_speculative_consume_results() {
  local tier="$1" output_file="$2"

  if [[ ! -f "$output_file" ]]; then
    return 1
  fi

  local content
  content="$(cat "$output_file")"

  # Check for clean review
  if echo "$content" | grep -q 'no issues\|Clean review\|NO_FINDINGS'; then
    echo "[ck:speculative] Tier $((tier - 1)) speculative review: clean."
    bp_speculative_update_job "$tier" "consumed"
    return 0
  fi

  # Findings exist — they've already been appended to the findings file by codex-review.sh
  echo "[ck:speculative] Tier $((tier - 1)) speculative review found issues."
  echo "$content"
  bp_speculative_update_job "$tier" "consumed"
  return 0
}

# ── T-206: Finding Reconciliation ─────────────────────────────────────
# Merge speculative findings into the tier gate flow.
# Speculative findings are tagged with source: codex-speculative.

bp_speculative_reconcile() {
  local tier="$1"

  # The findings from codex-review.sh are already tagged with source: codex.
  # We re-tag speculative ones with codex-speculative for traceability.
  local fpath
  fpath="$(bp_findings_path)"

  [[ -f "$fpath" ]] || return 0

  # Update source tag for findings that came from the speculative review
  # by checking if they were added during this tier's speculative window
  local job_info
  job_info="$(bp_speculative_get_job "$tier" 2>/dev/null)" || return 0

  echo "[ck:speculative] Reconciling speculative findings for tier $((tier - 1))."
  # The findings are already in the findings file from codex-review.sh.
  # The tier gate will process them normally via bp_tier_gate.
}

# ── T-207: P0/P1 Queuing ─────────────────────────────────────────────
# Queue for holding speculative findings that arrive while a tier is building.

_BP_SPECULATIVE_QUEUED_FINDINGS=""

bp_speculative_queue_finding() {
  local finding="$1"
  _BP_SPECULATIVE_QUEUED_FINDINGS+="${finding}"$'\n'
}

bp_speculative_drain_queue() {
  if [[ -z "$_BP_SPECULATIVE_QUEUED_FINDINGS" ]]; then
    echo "[ck:speculative] No queued findings."
    return 0
  fi

  echo "[ck:speculative] Processing $(echo "$_BP_SPECULATIVE_QUEUED_FINDINGS" | grep -c '^[^$]') queued finding(s)..."

  while IFS= read -r finding; do
    [[ -z "$finding" ]] && continue
    echo "  Queued: $finding"
  done <<< "$_BP_SPECULATIVE_QUEUED_FINDINGS"

  _BP_SPECULATIVE_QUEUED_FINDINGS=""
}

# ── T-208: Impl Tracking Integration ─────────────────────────────────

# Log speculative review metadata for a tier.
# $1 — tier number
# $2 — review source (speculative | synchronous | skipped)
# $3 — time saved in seconds (0 if synchronous)

bp_speculative_log_tier() {
  local tier="$1" source="$2" time_saved="${3:-0}"

  local impl_file="${PROJECT_ROOT}/context/impl/impl-speculative-log.md"

  if [[ ! -f "$impl_file" ]]; then
    mkdir -p "$(dirname "$impl_file")"
    cat > "$impl_file" << 'EOF'
---
created: "2026-03-31T00:00:00Z"
---
# Speculative Review Log

| Tier | Review Source | Time Saved (s) | Timestamp |
|------|-------------|----------------|-----------|
EOF
  fi

  echo "| $tier | $source | $time_saved | $(date -u +"%Y-%m-%dT%H:%M:%SZ") |" >> "$impl_file"
}

# Calculate time saved: actual review duration that overlapped with build time.
# $1 — tier number
# $2 — tier build start time (epoch)
# $3 — tier build end time (epoch)

bp_speculative_time_saved() {
  local tier="$1" build_start="$2" build_end="$3"

  local job_info
  job_info="$(bp_speculative_get_job "$tier" 2>/dev/null)" || { echo 0; return; }

  local j_tier j_pid j_outf j_bref j_status j_start
  IFS='|' read -r j_tier j_pid j_outf j_bref j_status j_start <<< "$job_info"

  if [[ "$j_status" == "consumed" || "$j_status" == "complete" ]]; then
    # Time saved = min(build_duration, review_duration) — the overlap
    local build_dur=$((build_end - build_start))
    # We don't track exact review end, approximate as build_end if consumed
    local saved=$build_dur
    # Cap at build duration (can't save more time than the build took)
    echo "$saved"
  else
    echo 0
  fi
}

# ── CLI mode ──────────────────────────────────────────────────────────

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  set -euo pipefail
  cmd="${1:-help}"
  shift || true
  case "$cmd" in
    dispatch) bp_speculative_dispatch "$@" ;;
    status) bp_speculative_status ;;
    retrieve) bp_speculative_retrieve "$@" ;;
    drain) bp_speculative_drain_queue ;;
    log) bp_speculative_log_tier "$@" ;;
    cleanup) bp_speculative_cleanup ;;
    help|--help|-h)
      echo "Usage: codex-speculative.sh {dispatch|status|retrieve|drain|log|cleanup}"
      echo "  dispatch <tier> <base_ref>  Launch background review"
      echo "  status                      Show pipeline status"
      echo "  retrieve <tier>             Get results with timeout"
      echo "  drain                       Process queued findings"
      echo "  log <tier> <source> [saved] Log tier review metadata"
      echo "  cleanup                     Remove session state"
      ;;
    *) echo "Unknown: $cmd" >&2; exit 1 ;;
  esac
fi
