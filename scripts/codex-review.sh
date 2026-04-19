#!/usr/bin/env bash
# codex-review.sh — T-006: Codex adversarial review invocation and finding parser
# Replaces MCP-based adversary invocation with Codex CLI delegation.
# Can be executed directly or sourced (exports bp_codex_review function).

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FINDINGS_FILE="$PROJECT_ROOT/context/impl/impl-review-findings.md"

# Source dependency scripts
if [[ -f "$SCRIPT_DIR/codex-detect.sh" ]]; then
  source "$SCRIPT_DIR/codex-detect.sh"
else
  codex_available=false
  CODEX_BINARY_AVAILABLE=false
fi

if [[ -f "$SCRIPT_DIR/codex-config.sh" ]]; then
  source "$SCRIPT_DIR/codex-config.sh"
else
  bp_config_get() { echo "${2:-}"; }
fi

# Note: custom review prompts were used with the old `codex --approval-mode`
# invocation. The current `codex review --base` subcommand uses its own
# built-in review logic. The prompt builder is kept for `codex exec` fallback.
_bp_build_review_prompt() {
  local caveman_active="false"
  if type bp_config_caveman_active &>/dev/null; then
    caveman_active="$(bp_config_caveman_active build)"
  fi

  if [[ "$caveman_active" == "true" ]]; then
    echo 'Senior engineer. Adversarial code review. Check diff for bugs, security holes, logic errors, spec violations. Each finding = one row in markdown table: Severity, File, Line, Description. Severity: P0 (critical) | P1 (high) | P2 (medium) | P3 (low). No issues found = output NO_FINDINGS alone.'
  else
    echo 'You are a senior engineer performing adversarial code review. Review the following diff for bugs, security issues, logic errors, and spec violations. For each finding output exactly one row in a markdown table with columns: Severity, File, Line, Description. Severity must be one of P0 (critical), P1 (high), P2 (medium), P3 (low). If no issues found, output exactly the word NO_FINDINGS on its own line and nothing else.'
  fi
}

# ── Main entry point ───────────────────────────────────────────────────

bp_codex_review() {
  local base_ref=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --base) base_ref="$2"; shift 2 ;;
      --help|-h) _codex_review_usage; return 0 ;;
      *) echo "[ck:review] Unknown argument: $1" >&2; return 1 ;;
    esac
  done

  # Check config
  local review_mode
  review_mode="$(bp_config_get codex_review auto)"
  if [[ "$review_mode" == "off" ]]; then
    echo "[ck:review] Codex review is disabled (codex_review=off). Skipping."
    return 0
  fi

  # Check availability — fall back when Codex unavailable
  if [[ "$codex_available" != "true" ]]; then
    echo "[ck:review] Codex is not available. Falling back to inspector-only review."
    return 0
  fi

  # Determine diff base
  if [[ -z "$base_ref" ]]; then
    base_ref="$(_detect_base_ref)"
  fi

  echo "[ck:review] Computing diff ${base_ref}...HEAD"

  local diff
  diff="$(git diff "${base_ref}...HEAD" 2>/dev/null || git diff "${base_ref}" HEAD 2>/dev/null || true)"

  if [[ -z "$diff" ]]; then
    echo "[ck:review] No diff found. Nothing to review."
    return 0
  fi

  local diff_lines
  diff_lines="$(echo "$diff" | wc -l | tr -d ' ')"
  echo "[ck:review] Diff is ${diff_lines} lines. Sending to Codex..."

  # Build Codex invocation — uses native `codex review` subcommand.
  # `codex review --base` handles diffing internally and uses its own
  # review prompt. Model comes from Codex config (~/.codex/config.toml),
  # not forced here — avoids model-availability errors across accounts.
  local codex_cmd=(codex review --base "$base_ref")

  if [[ "${BP_CODEX_DRY_RUN:-}" == "1" ]]; then
    echo "[ck:review] DRY RUN — would execute: ${codex_cmd[*]}"
    return 0
  fi

  local raw_output
  raw_output="$("${codex_cmd[@]}" 2>&1)" || {
    echo "[ck:review] Codex invocation failed. Falling back to inspector-only review."
    echo "[ck:review] Error: ${raw_output:0:500}"
    return 0
  }

  # Extract the review body from the Codex session transcript.
  # `codex review` outputs a full session log: header, tool calls (skill loading),
  # then the actual review as the final assistant ("codex") turn. We want only
  # the review body — everything after the last "codex\n" marker that follows
  # tool output, skipping the session preamble and skill-loading noise.
  local review_body
  review_body="$(_extract_codex_review_body "$raw_output")"

  if [[ -z "$review_body" ]]; then
    # Fallback: couldn't parse transcript structure, use full output
    review_body="$raw_output"
  fi

  # Check for clean review
  if echo "$review_body" | grep -qiE 'NO_FINDINGS|no issues|LGTM|looks good|no bugs|no problems'; then
    echo "[ck:review] Codex found no issues. Clean review."
    return 0
  fi

  echo "[ck:review] Parsing Codex findings..."

  # Try legacy table format first (P0-P3 severity rows)
  local findings
  findings="$(_parse_codex_findings "$review_body")"

  if [[ -z "$findings" ]]; then
    # Native codex review format — store full review body
    echo "[ck:review] Native review format detected. Storing findings."
    _append_raw_review "$review_body"
    echo ""
    echo "[ck:review] === Codex Review Output ==="
    echo "$review_body"
    echo "[ck:review] === End of Review ==="
    echo "[ck:review] Findings appended to $FINDINGS_FILE"
    return 0
  fi

  _append_findings_to_file "$findings"

  echo ""
  echo "[ck:review] === Codex Adversarial Review Findings ==="
  echo "$findings"
  echo "[ck:review] === End of Findings ==="
  echo "[ck:review] Findings appended to $FINDINGS_FILE"

  return 0
}

# ── Helpers ────────────────────────────────────────────────────────────

_codex_review_usage() {
  cat <<EOF
Usage: codex-review.sh [--base <ref>]

Perform adversarial code review using Codex CLI.

Options:
  --base <ref>    Git ref to diff against (default: auto-detect)
  --help, -h      Show this help

Environment:
  BP_CODEX_DRY_RUN=1    Print the command without executing
EOF
}

_detect_base_ref() {
  local worktree_base
  worktree_base="$(git rev-parse --abbrev-ref @{upstream} 2>/dev/null || true)"
  if [[ -n "$worktree_base" ]]; then echo "$worktree_base"; return; fi

  for candidate in main master develop; do
    if git rev-parse --verify "$candidate" &>/dev/null; then
      echo "$candidate"; return
    fi
  done

  echo "HEAD~10"
}

_parse_codex_findings() {
  local raw="$1"
  local output=""
  local finding_num

  finding_num="$(_next_finding_number)"

  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    [[ "$line" =~ ^[[:space:]]*\|[[:space:]]*-+ ]] && continue
    [[ "$line" =~ ^[[:space:]]*\|[[:space:]]*Severity ]] && continue

    if echo "$line" | grep -qE '\|[[:space:]]*P[0-3]'; then
      local severity file lineno description

      severity="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $2); print $2}')"
      file="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $3); print $3}')"
      lineno="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $4); print $4}')"
      description="$(echo "$line" | awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $5); print $5}')"

      severity="$(echo "$severity" | tr -d '\`' | xargs)"
      file="$(echo "$file" | tr -d '\`' | xargs)"
      lineno="$(echo "$lineno" | tr -d '\`' | xargs)"
      description="$(echo "$description" | tr -d '\`' | xargs)"

      local file_ref="$file"
      if [[ -n "$lineno" && "$lineno" != "-" && "$lineno" != "N/A" ]]; then
        file_ref="${file}:L${lineno}"
      fi

      local fid
      fid="$(printf 'F-%03d' "$finding_num")"

      output+="| ${fid}: ${description} (source: codex) | ${severity} | ${file_ref} | NEW | — |"$'\n'
      finding_num=$((finding_num + 1))
    fi
  done <<< "$raw"

  echo "$output"
}

_next_finding_number() {
  if [[ ! -f "$FINDINGS_FILE" ]]; then echo 1; return; fi

  local max
  max="$(grep -oE 'F-[0-9]+' "$FINDINGS_FILE" 2>/dev/null | sed 's/F-//' | sort -n | tail -1)"

  if [[ -n "$max" ]]; then
    echo $((10#$max + 1))
  else
    echo 1
  fi
}

_extract_codex_review_body() {
  # Codex session transcript format (simplified):
  #   <header block>
  #   user
  #   <user message>
  #   exec
  #   <tool call + output>
  #   codex              ← the assistant's review response starts here
  #   <actual review>
  #
  # We want everything after the LAST "^codex$" line, which is the
  # final assistant turn containing the actual review findings.
  local raw="$1"

  # Find the last occurrence of a line that is exactly "codex"
  # (the assistant turn marker in the transcript)
  local last_codex_line
  last_codex_line="$(echo "$raw" | grep -n '^codex$' | tail -1 | cut -d: -f1)"

  if [[ -z "$last_codex_line" ]]; then
    # No "codex" turn marker found — can't parse transcript
    echo ""
    return
  fi

  # Extract everything after that line
  echo "$raw" | tail -n +"$((last_codex_line + 1))"
}

_append_raw_review() {
  local raw="$1"
  mkdir -p "$(dirname "$FINDINGS_FILE")"

  if [[ ! -f "$FINDINGS_FILE" ]]; then
    cat > "$FINDINGS_FILE" << 'HEADER'
# Review Findings

| Finding | Severity | File | Status | Task |
|---------|----------|------|--------|------|
HEADER
  fi

  # Append native review output as a fenced block (no truncation)
  {
    echo ""
    echo "### Codex Native Review — $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
    echo ""
    echo '```'
    echo "$raw"
    echo '```'
  } >> "$FINDINGS_FILE"
}

_append_findings_to_file() {
  local findings="$1"

  mkdir -p "$(dirname "$FINDINGS_FILE")"

  if [[ ! -f "$FINDINGS_FILE" ]]; then
    cat > "$FINDINGS_FILE" << 'HEADER'
# Review Findings

| Finding | Severity | File | Status | Task |
|---------|----------|------|--------|------|
HEADER
  fi

  echo "$findings" >> "$FINDINGS_FILE"
}

# ── Direct execution ───────────────────────────────────────────────────

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  set -euo pipefail
  bp_codex_review "$@"
fi
