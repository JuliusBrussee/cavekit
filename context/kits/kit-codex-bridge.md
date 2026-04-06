---
created: "2026-03-31T00:00:00Z"
last_edited: "2026-03-31T00:00:00Z"
---

# Cavekit: Codex Bridge

## Scope
Everything related to detecting, configuring, and communicating with the Codex Claude Code plugin (`openai/codex-plugin-cc`) from within Cavekit. This domain makes Codex available as an adversary and provides a standalone review command.

## Requirements

### R1: Plugin Detection
Detect whether the Codex Claude Code plugin is installed and the `codex` binary is available.
- [ ] Check for Codex plugin presence at Cavekit command entry (not a persistent daemon — a one-shot check)
- [ ] Check `codex` binary is on PATH and responsive (`codex --version` or equivalent)
- [ ] Expose a boolean `codex_available` flag consumable by other Cavekit components (e.g., as a shell function or variable in the build scripts)
- [ ] When Codex is absent, emit a one-time suggestion to install it, then proceed with existing inspector-only review

### R2: Configuration
User-configurable settings for Codex integration behavior.
- [ ] Setting `codex_review` with values `"auto" | "off"` — `auto` enables Codex review when detected, `off` disables entirely
- [ ] Setting `codex_model` to override the model used for reviews (defaults to Codex's own config)
- [ ] Setting `codex_effort` to override reasoning effort (defaults to Codex's own config)
- [ ] Settings stored in Cavekit's existing config mechanism (project-level `.cavekit/config` or user-level)
- [ ] Default: `codex_review = "auto"`

### R3: Peer-Review-Loop Replacement
When Codex is available, it becomes the sole adversary in Cavekit's peer-review-loop, replacing the MCP-based adversary.
- [ ] Replace MCP-based adversary invocation with `/codex:adversarial-review` delegation
- [ ] Pass the current diff context (worktree base to HEAD) to Codex
- [ ] Parse Codex review output into Cavekit's existing finding format (severity P0-P3, file, line, description)
- [ ] Fall back to existing inspector-only review when Codex is unavailable
- [ ] Findings tagged with `source: codex` in `impl-review-findings.md`

### R4: Standalone Review Command
A `/ck:judge` command that invokes Codex adversarial review on demand.
- [ ] Default target: current build tier's diff against worktree base
- [ ] Accept `--base <ref>` override for custom diff range
- [ ] Output findings in Cavekit's standard finding format to stdout
- [ ] Append findings to `impl-review-findings.md` with source tagged as `codex`
- [ ] Error gracefully with a clear message if Codex is not installed

## Out of Scope
- Codex plugin installation (that's `/codex:setup`'s job)
- Build-loop orchestration and tier gating (that's Tier Gate's job)
- Modifying the Codex plugin itself
- MCP-based adversary as a fallback option (removed — Codex replaces it entirely when present)

## Cross-References
- See also: cavekit-tier-gate.md (depends on R1 for detection and R3 for finding format)
- See also: cavekit-command-gate.md (depends on R1 for detection and R2 for shared config mechanism)
- See also: cavekit-speculative-review.md (depends on R1 for detection, R3 for review invocation)
- See also: cavekit-draft-challenge.md (depends on R1 for detection, R3 for review invocation)

## Changelog
- 2026-03-31: Initial draft
