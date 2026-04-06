#!/usr/bin/env bash
# codex-design-challenge.sh — Design Challenge: Codex adversarial cavekit review
# T-301: Design challenge prompt template
# T-302: Challenge output parser
#
# Source this file to get bp_design_challenge / bp_parse_challenge_findings.
# Execute directly to run a challenge against kits in context/kits/.

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

# Guard against double-sourcing
[[ -n "${_BP_DESIGN_CHALLENGE_LOADED:-}" ]] && { return 0 2>/dev/null || true; }
_BP_DESIGN_CHALLENGE_LOADED=1

# ── T-301: Design Challenge Prompt Template ───────────────────────────

_bp_design_challenge_prompt() {
  local caveman_active="false"
  if type bp_config_caveman_active &>/dev/null; then
    caveman_active="$(bp_config_caveman_active draft)"
  fi

  if [[ "$caveman_active" == "true" ]]; then
    cat <<'PROMPT'
Senior architect. Adversarial design review of cavekit specs. CHALLENGE design, not rubber-stamp.

Review all kits as whole system. Design-level concerns only:

1. **Domain Decomposition** — boundaries right? Over/under-scoped? Better decomposition possible?
2. **Requirement Coverage** — missing reqs? Gaps between domains? Cross-refs cover all interactions?
3. **Ambiguity** — criteria vague? Checkbox reqs? Implementer know exactly what to build?
4. **Scope** — domains over/under-scoped? Implicit assumptions?
5. **Cross-Domain Coherence** — domains fit together? Contradictions? Hidden circular deps?

Rules: NO impl feedback. MUST propose alternative decomposition if better exists. Reference cavekit files + R-numbers.

Output: one row per finding in markdown table: Category, Severity, Cavekit, Requirement, Description
Category: decomposition|coverage|ambiguity|scope|assumption
Severity: critical|advisory
No issues = NO_ISSUES

## Kits to Review

PROMPT
  else
    cat <<'PROMPT'
You are a senior software architect performing an adversarial design review of cavekit specifications. Your job is to CHALLENGE the design, not rubber-stamp it.

Review all kits as a whole system. Focus exclusively on design-level concerns:

1. **Domain Decomposition Quality**
   - Are domain boundaries drawn at the right places?
   - Is any domain doing too much (over-scoped) or too little (under-scoped)?
   - Would a different decomposition reduce coupling or improve cohesion?

2. **Requirement Coverage**
   - Are there missing requirements that the system clearly needs?
   - Are there gaps between domains where functionality falls through the cracks?
   - Do cross-references cover all domain interactions?

3. **Ambiguity in Acceptance Criteria**
   - Are any acceptance criteria vague enough to be interpreted two different ways?
   - Are any criteria technically testable but practically meaningless ("checkbox requirements")?
   - Would an implementer know exactly what to build from each requirement?

4. **Scope Assessment**
   - Is any domain over-scoped (trying to do too much for one implementation unit)?
   - Is any domain under-scoped (too thin to be worth its own domain)?
   - Are there implicit assumptions that should be made explicit?

5. **Cross-Domain Coherence**
   - Do the domains fit together as a coherent system?
   - Are there contradictions between domains?
   - Is the dependency graph sound (no hidden circular dependencies)?

## Rules
- Do NOT provide implementation-level feedback (no framework suggestions, no file path opinions, no API design)
- You MUST propose at least one alternative decomposition if you can identify a better one
- Focus on issues that would cause real problems during implementation
- Be specific: reference cavekit files and requirement numbers

## Output Format

For each finding, output exactly one row in a markdown table with columns:
  Category, Severity, Cavekit, Requirement, Description

Category must be one of: decomposition, coverage, ambiguity, scope, assumption
Severity must be one of: critical, advisory

If you find no issues at all, output exactly: NO_ISSUES

## Kits to Review

PROMPT
  fi
}

# ── T-302: Challenge Output Parser ────────────────────────────────────

# Parse Codex design challenge output into structured findings.
# Input: raw Codex output (stdin or $1)
# Output: structured findings, one per line:
#   CATEGORY|SEVERITY|CAVEKIT|REQUIREMENT|DESCRIPTION
#
# Also sets:
#   _BP_CHALLENGE_CRITICAL_COUNT
#   _BP_CHALLENGE_ADVISORY_COUNT

bp_parse_challenge_findings() {
  local raw="${1:-$(cat)}"

  _BP_CHALLENGE_CRITICAL_COUNT=0
  _BP_CHALLENGE_ADVISORY_COUNT=0

  if echo "$raw" | grep -qi 'NO_ISSUES'; then
    return 0
  fi

  local findings=""

  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    # Skip table header and separator rows
    [[ "$line" =~ ^[[:space:]]*\|[[:space:]]*-+ ]] && continue
    [[ "$line" =~ ^[[:space:]]*\|[[:space:]]*Category ]] && continue

    # Match rows with our expected categories
    if echo "$line" | grep -qE '\|\s*(decomposition|coverage|ambiguity|scope|assumption)'; then
      local category severity cavekit requirement description

      category="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $2); print $2}')"
      severity="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $3); print $3}')"
      cavekit="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $4); print $4}')"
      requirement="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $5); print $5}')"
      description="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $6); print $6}')"

      # Clean backticks
      category="$(echo "$category" | tr -d '\`' | xargs)"
      severity="$(echo "$severity" | tr -d '\`' | xargs)"
      cavekit="$(echo "$cavekit" | tr -d '\`' | xargs)"
      requirement="$(echo "$requirement" | tr -d '\`' | xargs)"
      description="$(echo "$description" | tr -d '\`' | xargs)"

      # Normalize severity
      severity="$(echo "$severity" | tr '[:upper:]' '[:lower:]')"
      case "$severity" in
        critical) _BP_CHALLENGE_CRITICAL_COUNT=$((_BP_CHALLENGE_CRITICAL_COUNT + 1)) ;;
        advisory) _BP_CHALLENGE_ADVISORY_COUNT=$((_BP_CHALLENGE_ADVISORY_COUNT + 1)) ;;
        *) severity="advisory"; _BP_CHALLENGE_ADVISORY_COUNT=$((_BP_CHALLENGE_ADVISORY_COUNT + 1)) ;;
      esac

      findings+="${category}|${severity}|${cavekit}|${requirement}|${description}"$'\n'
    fi
  done <<< "$raw"

  if [[ -n "$findings" ]]; then
    echo "$findings" | grep -v '^$'
  fi
}

# ── bp_design_challenge ───────────────────────────────────────────────
# Main entry point: send kits to Codex for design challenge.
#
# Arguments:
#   --kits-dir <path>  Directory containing kits (default: context/kits/)
#
# Returns:
#   0 — no critical issues (clean or advisory-only)
#   1 — critical issues found (findings printed to stdout)
#   2 — Codex unavailable or invocation failed (graceful skip)

bp_design_challenge() {
  local kits_dir="${PROJECT_ROOT}/context/kits"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --kits-dir) kits_dir="$2"; shift 2 ;;
      --help|-h) _design_challenge_usage; return 0 ;;
      *) echo "[ck:design-challenge] Unknown argument: $1" >&2; return 2 ;;
    esac
  done

  # Check availability
  if [[ "$codex_available" != "true" ]]; then
    echo "[ck:design-challenge] Codex unavailable — skipping design challenge."
    return 2
  fi

  local review_mode
  review_mode="$(bp_config_get codex_review auto)"
  if [[ "$review_mode" == "off" ]]; then
    echo "[ck:design-challenge] Codex review disabled (codex_review=off). Skipping."
    return 2
  fi

  # Gather kits
  if [[ ! -d "$kits_dir" ]]; then
    echo "[ck:design-challenge] No kits directory at $kits_dir" >&2
    return 2
  fi

  local cavekit_content=""
  local file_count=0
  for f in "$kits_dir"/cavekit-*.md; do
    [[ -f "$f" ]] || continue
    cavekit_content+="--- FILE: $(basename "$f") ---"$'\n'
    cavekit_content+="$(cat "$f")"$'\n\n'
    file_count=$((file_count + 1))
  done

  if [[ $file_count -eq 0 ]]; then
    echo "[ck:design-challenge] No cavekit files found in $kits_dir" >&2
    return 2
  fi

  echo "[ck:design-challenge] Sending $file_count cavekit(s) to Codex for design challenge..."

  # Build the full prompt
  local full_prompt
  full_prompt="$(_bp_design_challenge_prompt)${cavekit_content}"

  # Build Codex invocation
  local model
  model="$(bp_config_get codex_model o4-mini)"
  local start_time
  start_time="$(date +%s)"

  local codex_cmd=(codex --approval-mode full-auto --model "$model" --quiet -p "$full_prompt")

  if [[ "${BP_CODEX_DRY_RUN:-}" == "1" ]]; then
    echo "[ck:design-challenge] DRY RUN — would send $file_count kits to Codex"
    return 0
  fi

  local raw_output
  raw_output="$(echo "" | "${codex_cmd[@]}" 2>&1)" || {
    echo "[ck:design-challenge] Codex invocation failed. Skipping design challenge."
    echo "[ck:design-challenge] Error: ${raw_output:0:500}"
    return 2
  }

  local end_time duration
  end_time="$(date +%s)"
  duration=$((end_time - start_time))

  # Parse findings
  local findings
  findings="$(bp_parse_challenge_findings "$raw_output")"

  if [[ -z "$findings" ]]; then
    echo "[ck:design-challenge] Codex found no design issues. Clean review. (${duration}s)"
    return 0
  fi

  echo "[ck:design-challenge] Challenge complete in ${duration}s: ${_BP_CHALLENGE_CRITICAL_COUNT} critical, ${_BP_CHALLENGE_ADVISORY_COUNT} advisory"

  # Output findings
  echo ""
  echo "=== Design Challenge Findings ==="
  echo "| Category | Severity | Cavekit | Requirement | Description |"
  echo "|----------|----------|-----------|-------------|-------------|"
  while IFS='|' read -r cat sev bp req desc; do
    [[ -z "$cat" ]] && continue
    echo "| $cat | $sev | $bp | $req | $desc |"
  done <<< "$findings"
  echo "=== End of Findings ==="

  if [[ $_BP_CHALLENGE_CRITICAL_COUNT -gt 0 ]]; then
    return 1
  fi

  return 0
}

# ── T-304: Advisory Findings Collector ─────────────────────────────────
# Formats findings for user-facing presentation in the draft flow.
# Separates critical (for auto-fix) from advisory (for user review).
#
# Input: findings from bp_parse_challenge_findings (pipe-delimited lines)
# Sets:
#   _BP_CHALLENGE_CRITICAL_FINDINGS  — newline-separated critical findings
#   _BP_CHALLENGE_ADVISORY_FINDINGS  — newline-separated advisory findings

bp_collect_challenge_findings() {
  local findings="${1:-$(cat)}"

  _BP_CHALLENGE_CRITICAL_FINDINGS=""
  _BP_CHALLENGE_ADVISORY_FINDINGS=""

  while IFS='|' read -r cat sev bp req desc; do
    [[ -z "$cat" ]] && continue
    local entry="${cat}|${sev}|${bp}|${req}|${desc}"
    case "$sev" in
      critical) _BP_CHALLENGE_CRITICAL_FINDINGS+="${entry}"$'\n' ;;
      *)        _BP_CHALLENGE_ADVISORY_FINDINGS+="${entry}"$'\n' ;;
    esac
  done <<< "$findings"
}

# Format advisory findings as a markdown block for user review gate.
# Call after bp_collect_challenge_findings.
# Output: markdown text suitable for display in the draft flow.

bp_format_advisory_for_user() {
  if [[ -z "$_BP_CHALLENGE_ADVISORY_FINDINGS" ]]; then
    echo "No advisory findings from Codex design challenge."
    return 0
  fi

  echo "### Codex Design Challenge — Advisory Findings"
  echo ""
  echo "These findings are informational — Codex flagged them as worth considering but not blocking."
  echo ""
  echo "| Category | Cavekit | Requirement | Finding |"
  echo "|----------|-----------|-------------|---------|"

  while IFS='|' read -r cat sev bp req desc; do
    [[ -z "$cat" ]] && continue
    echo "| $cat | $bp | $req | $desc |"
  done <<< "$_BP_CHALLENGE_ADVISORY_FINDINGS"
}

# Format critical findings for auto-fix processing.
# Call after bp_collect_challenge_findings.
# Output: one finding per line in format: CAVEKIT|REQUIREMENT|CATEGORY|DESCRIPTION

bp_format_critical_for_fix() {
  if [[ -z "$_BP_CHALLENGE_CRITICAL_FINDINGS" ]]; then
    return 0
  fi

  while IFS='|' read -r cat sev bp req desc; do
    [[ -z "$cat" ]] && continue
    echo "${bp}|${req}|${cat}|${desc}"
  done <<< "$_BP_CHALLENGE_CRITICAL_FINDINGS"
}

# ── Helpers ───────────────────────────────────────────────────────────

_design_challenge_usage() {
  cat <<EOF
Usage: codex-design-challenge.sh [--kits-dir <path>]

Send kits to Codex for adversarial design review.

Options:
  --kits-dir <path>  Cavekit directory (default: context/kits/)
  --help, -h               Show this help

Environment:
  BP_CODEX_DRY_RUN=1       Print the command without executing
EOF
}

# ── T-305: Auto-Fix and Re-Challenge Loop ─────────────────────────────
# Orchestrates the challenge-fix-rechallenge cycle.
#
# Arguments:
#   --kits-dir <path>  Cavekit directory
#   --max-cycles <N>         Maximum challenge-fix cycles (default: 2)
#
# Returns:
#   0 — no critical issues remain
#   1 — critical issues remain after max cycles (advisory + remaining criticals printed)
#   2 — Codex unavailable (skip)
#
# The caller (draft flow) is responsible for implementing fixes between cycles.
# This function outputs AWAITING_FIXES when fixes are needed, along with the
# critical findings in structured format for the caller to process.

bp_design_challenge_cycle() {
  local kits_dir="${PROJECT_ROOT}/context/kits"
  local max_cycles=2
  local cycle=0

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --kits-dir) kits_dir="$2"; shift 2 ;;
      --max-cycles) max_cycles="$2"; shift 2 ;;
      *) shift ;;
    esac
  done

  while (( cycle < max_cycles )); do
    cycle=$((cycle + 1))
    echo "[ck:design-challenge] Challenge cycle ${cycle}/${max_cycles}"

    # Run the challenge
    local challenge_output
    challenge_output="$(bp_design_challenge --kits-dir "$kits_dir" 2>&1)"
    local rc=$?

    echo "$challenge_output"

    case $rc in
      0)
        # No critical issues — collect advisory for user display
        echo "[ck:design-challenge] Cycle ${cycle}: No critical issues."
        return 0
        ;;
      2)
        # Codex unavailable
        return 2
        ;;
      1)
        # Critical issues found — need fixes
        # Re-parse the raw findings from the challenge output
        local table_lines
        table_lines="$(echo "$challenge_output" | sed -n '/=== Design Challenge Findings ===/,/=== End of Findings ===/p' | grep -E '^\|' | grep -vE '^\| Category' | grep -vE '^\|--')"

        if [[ -z "$table_lines" ]]; then
          echo "[ck:design-challenge] Could not extract findings from output."
          return 1
        fi

        # Parse into our structured format
        local parsed=""
        while IFS= read -r row; do
          local cat sev bp req desc
          cat="$(echo "$row" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $2); print $2}')"
          sev="$(echo "$row" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $3); print $3}')"
          bp="$(echo "$row" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $4); print $4}')"
          req="$(echo "$row" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $5); print $5}')"
          desc="$(echo "$row" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $6); print $6}')"
          parsed+="${cat}|${sev}|${bp}|${req}|${desc}"$'\n'
        done <<< "$table_lines"

        bp_collect_challenge_findings "$parsed"

        if [[ $_BP_CHALLENGE_CRITICAL_COUNT -eq 0 ]]; then
          echo "[ck:design-challenge] Cycle ${cycle}: Only advisory findings remain."
          return 0
        fi

        if (( cycle < max_cycles )); then
          echo "[ck:design-challenge] ${_BP_CHALLENGE_CRITICAL_COUNT} critical finding(s) need fixes."
          echo "CRITICAL_FIXES:"
          bp_format_critical_for_fix
          echo "AWAITING_FIXES"
          # Caller implements fixes and re-runs the cycle
          return 3  # Signal: fixes needed
        fi
        ;;
    esac
  done

  # Exhausted max cycles — report remaining
  echo "[ck:design-challenge] WARNING: ${_BP_CHALLENGE_CRITICAL_COUNT} critical finding(s) remain after ${max_cycles} cycles."
  echo "[ck:design-challenge] Presenting remaining findings to user for judgment."

  if [[ -n "${_BP_CHALLENGE_CRITICAL_FINDINGS:-}" ]]; then
    echo ""
    echo "### Remaining Critical Findings (require user judgment)"
    echo ""
    echo "| Category | Cavekit | Requirement | Finding |"
    echo "|----------|-----------|-------------|---------|"
    while IFS='|' read -r cat sev bp req desc; do
      [[ -z "$cat" ]] && continue
      echo "| $cat | $bp | $req | $desc |"
    done <<< "$_BP_CHALLENGE_CRITICAL_FINDINGS"
  fi

  if [[ -n "${_BP_CHALLENGE_ADVISORY_FINDINGS:-}" ]]; then
    bp_format_advisory_for_user
  fi

  return 1
}

# ── T-306: Draft Flow Integration Point ───────────────────────────────
# Entry point for the /ck:sketch command to call after Step 8 (cavekit-reviewer).
# Runs the design challenge and returns results for insertion before Step 9 (user gate).
#
# Arguments:
#   --kits-dir <path>  Cavekit directory
#
# Output:
#   Sets BP_CHALLENGE_ADVISORY_OUTPUT — markdown text to show user at Step 9
#   Sets BP_CHALLENGE_DURATION — seconds the challenge took
#
# Returns:
#   0 — challenge passed (clean or advisory-only), proceed to user gate
#   1 — critical issues remain after auto-fix, user must decide
#   2 — skipped (Codex unavailable), proceed to user gate without challenge

bp_draft_challenge_hook() {
  local kits_dir="${PROJECT_ROOT}/context/kits"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --kits-dir) kits_dir="$2"; shift 2 ;;
      *) shift ;;
    esac
  done

  BP_CHALLENGE_ADVISORY_OUTPUT=""
  BP_CHALLENGE_DURATION=0

  local start_time
  start_time="$(date +%s)"

  echo "[ck:sketch] Running Codex design challenge..."

  local result
  result="$(bp_design_challenge_cycle --kits-dir "$kits_dir" 2>&1)"
  local rc=$?

  local end_time
  end_time="$(date +%s)"
  BP_CHALLENGE_DURATION=$((end_time - start_time))

  echo "$result"

  case $rc in
    0)
      # Collect any advisory findings for user gate
      BP_CHALLENGE_ADVISORY_OUTPUT="$(bp_format_advisory_for_user 2>/dev/null || true)"
      echo "[ck:sketch] Design challenge passed (${BP_CHALLENGE_DURATION}s)."
      return 0
      ;;
    2)
      echo "[ck:sketch] Design challenge skipped — Codex unavailable (${BP_CHALLENGE_DURATION}s)."
      return 2
      ;;
    *)
      # Critical findings remain — let caller decide
      BP_CHALLENGE_ADVISORY_OUTPUT="$(bp_format_advisory_for_user 2>/dev/null || true)"
      echo "[ck:sketch] Design challenge has unresolved critical findings (${BP_CHALLENGE_DURATION}s)."
      return 1
      ;;
  esac
}

# ── CLI mode ──────────────────────────────────────────────────────────

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  set -euo pipefail
  bp_design_challenge "$@"
fi
