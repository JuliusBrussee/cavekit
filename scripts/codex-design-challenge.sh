#!/usr/bin/env bash
# codex-design-challenge.sh — Design Challenge: Codex adversarial blueprint review
# T-301: Design challenge prompt template
# T-302: Challenge output parser
#
# Source this file to get bp_design_challenge / bp_parse_challenge_findings.
# Execute directly to run a challenge against blueprints in context/blueprints/.

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
  cat <<'PROMPT'
You are a senior software architect performing an adversarial design review of blueprint specifications. Your job is to CHALLENGE the design, not rubber-stamp it.

Review all blueprints as a whole system. Focus exclusively on design-level concerns:

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
- Be specific: reference blueprint files and requirement numbers

## Output Format

For each finding, output exactly one row in a markdown table with columns:
  Category, Severity, Blueprint, Requirement, Description

Category must be one of: decomposition, coverage, ambiguity, scope, assumption
Severity must be one of: critical, advisory

If you find no issues at all, output exactly: NO_ISSUES

## Blueprints to Review

PROMPT
}

# ── T-302: Challenge Output Parser ────────────────────────────────────

# Parse Codex design challenge output into structured findings.
# Input: raw Codex output (stdin or $1)
# Output: structured findings, one per line:
#   CATEGORY|SEVERITY|BLUEPRINT|REQUIREMENT|DESCRIPTION
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
      local category severity blueprint requirement description

      category="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $2); print $2}')"
      severity="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $3); print $3}')"
      blueprint="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $4); print $4}')"
      requirement="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $5); print $5}')"
      description="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $6); print $6}')"

      # Clean backticks
      category="$(echo "$category" | tr -d '\`' | xargs)"
      severity="$(echo "$severity" | tr -d '\`' | xargs)"
      blueprint="$(echo "$blueprint" | tr -d '\`' | xargs)"
      requirement="$(echo "$requirement" | tr -d '\`' | xargs)"
      description="$(echo "$description" | tr -d '\`' | xargs)"

      # Normalize severity
      severity="$(echo "$severity" | tr '[:upper:]' '[:lower:]')"
      case "$severity" in
        critical) _BP_CHALLENGE_CRITICAL_COUNT=$((_BP_CHALLENGE_CRITICAL_COUNT + 1)) ;;
        advisory) _BP_CHALLENGE_ADVISORY_COUNT=$((_BP_CHALLENGE_ADVISORY_COUNT + 1)) ;;
        *) severity="advisory"; _BP_CHALLENGE_ADVISORY_COUNT=$((_BP_CHALLENGE_ADVISORY_COUNT + 1)) ;;
      esac

      findings+="${category}|${severity}|${blueprint}|${requirement}|${description}"$'\n'
    fi
  done <<< "$raw"

  if [[ -n "$findings" ]]; then
    echo "$findings" | grep -v '^$'
  fi
}

# ── bp_design_challenge ───────────────────────────────────────────────
# Main entry point: send blueprints to Codex for design challenge.
#
# Arguments:
#   --blueprints-dir <path>  Directory containing blueprints (default: context/blueprints/)
#
# Returns:
#   0 — no critical issues (clean or advisory-only)
#   1 — critical issues found (findings printed to stdout)
#   2 — Codex unavailable or invocation failed (graceful skip)

bp_design_challenge() {
  local blueprints_dir="${PROJECT_ROOT}/context/blueprints"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --blueprints-dir) blueprints_dir="$2"; shift 2 ;;
      --help|-h) _design_challenge_usage; return 0 ;;
      *) echo "[bp:design-challenge] Unknown argument: $1" >&2; return 2 ;;
    esac
  done

  # Check availability
  if [[ "$codex_available" != "true" ]]; then
    echo "[bp:design-challenge] Codex unavailable — skipping design challenge."
    return 2
  fi

  local review_mode
  review_mode="$(bp_config_get codex_review auto)"
  if [[ "$review_mode" == "off" ]]; then
    echo "[bp:design-challenge] Codex review disabled (codex_review=off). Skipping."
    return 2
  fi

  # Gather blueprints
  if [[ ! -d "$blueprints_dir" ]]; then
    echo "[bp:design-challenge] No blueprints directory at $blueprints_dir" >&2
    return 2
  fi

  local blueprint_content=""
  local file_count=0
  for f in "$blueprints_dir"/blueprint-*.md; do
    [[ -f "$f" ]] || continue
    blueprint_content+="--- FILE: $(basename "$f") ---"$'\n'
    blueprint_content+="$(cat "$f")"$'\n\n'
    file_count=$((file_count + 1))
  done

  if [[ $file_count -eq 0 ]]; then
    echo "[bp:design-challenge] No blueprint files found in $blueprints_dir" >&2
    return 2
  fi

  echo "[bp:design-challenge] Sending $file_count blueprint(s) to Codex for design challenge..."

  # Build the full prompt
  local full_prompt
  full_prompt="$(_bp_design_challenge_prompt)${blueprint_content}"

  # Build Codex invocation
  local model
  model="$(bp_config_get codex_model o4-mini)"
  local start_time
  start_time="$(date +%s)"

  local codex_cmd=(codex --approval-mode full-auto --model "$model" --quiet -p "$full_prompt")

  if [[ "${BP_CODEX_DRY_RUN:-}" == "1" ]]; then
    echo "[bp:design-challenge] DRY RUN — would send $file_count blueprints to Codex"
    return 0
  fi

  local raw_output
  raw_output="$(echo "" | "${codex_cmd[@]}" 2>&1)" || {
    echo "[bp:design-challenge] Codex invocation failed. Skipping design challenge."
    echo "[bp:design-challenge] Error: ${raw_output:0:500}"
    return 2
  }

  local end_time duration
  end_time="$(date +%s)"
  duration=$((end_time - start_time))

  # Parse findings
  local findings
  findings="$(bp_parse_challenge_findings "$raw_output")"

  if [[ -z "$findings" ]]; then
    echo "[bp:design-challenge] Codex found no design issues. Clean review. (${duration}s)"
    return 0
  fi

  echo "[bp:design-challenge] Challenge complete in ${duration}s: ${_BP_CHALLENGE_CRITICAL_COUNT} critical, ${_BP_CHALLENGE_ADVISORY_COUNT} advisory"

  # Output findings
  echo ""
  echo "=== Design Challenge Findings ==="
  echo "| Category | Severity | Blueprint | Requirement | Description |"
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

# ── Helpers ───────────────────────────────────────────────────────────

_design_challenge_usage() {
  cat <<EOF
Usage: codex-design-challenge.sh [--blueprints-dir <path>]

Send blueprints to Codex for adversarial design review.

Options:
  --blueprints-dir <path>  Blueprint directory (default: context/blueprints/)
  --help, -h               Show this help

Environment:
  BP_CODEX_DRY_RUN=1       Print the command without executing
EOF
}

# ── CLI mode ──────────────────────────────────────────────────────────

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  set -euo pipefail
  bp_design_challenge "$@"
fi
