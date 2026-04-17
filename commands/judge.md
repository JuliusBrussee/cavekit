---
name: ck-judge
description: "Invoke Codex adversarial review on demand — diffs current tier against worktree base and outputs findings"
argument-hint: "[--base REF]"
allowed-tools: ["Bash(${CLAUDE_PLUGIN_ROOT}/scripts/codex-review.sh*)"]
---

> **Note:** `/bp:codex-review`, `/ck:codex-review`, `/bp:judge` are deprecated aliases. Use `/ck:judge` instead.

# /ck:judge — Codex Adversarial Review

Run an on-demand adversarial code review using the Codex CLI. This sends the current diff to Codex for analysis of bugs, security issues, logic errors, and spec violations.

## Execution

Run the review script, forwarding any arguments the user provided: `"${CLAUDE_PLUGIN_ROOT}/scripts/codex-review.sh" $ARGUMENTS`.

## Behavior

- **Default target**: diffs the current build tier's changes against the worktree base (auto-detected from upstream tracking branch, or falls back to `main`/`master`/`develop`).
- **`--base <ref>`**: override the diff base to any git ref (branch, tag, or commit SHA).
- Findings are printed to stdout in Cavekit's standard finding format (`F-NNN` numbered, severity `P0`–`P3`, source tagged as `codex`).
- Findings are automatically appended to `context/impl/impl-review-findings.md`.
- If Codex is not installed or not available, the script prints a clear message and exits gracefully — no error is thrown.

## After the script completes

1. **If findings were reported**: summarize them for the user — count by severity, highlight any P0/P1 items, and note the findings file path.
2. **If no findings**: confirm a clean review to the user.
3. **If Codex was unavailable**: inform the user that Codex CLI is required and suggest installing it (`npm install -g @openai/codex` or equivalent).
