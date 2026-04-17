---
created: "2026-03-31T00:00:00Z"
last_edited: "2026-03-31T00:00:00Z"
---

# Cavekit: Command Gate

## Scope
A PreToolUse hook that intercepts Bash tool calls, classifies command safety using Codex (gpt-5.4-mini), and blocks unsafe commands before execution. Integrates with Claude Code's existing allow/block permission system and caches verdicts by normalized command pattern.

## Requirements

### R1: PreToolUse Hook
Intercept Bash tool calls before execution via Claude Code's PreToolUse hook system.
- [ ] Register a PreToolUse hook that matches on Bash tool invocations
- [ ] Extract the command string from the tool call parameters
- [ ] Return one of: approve (allow execution), block (prevent execution with reason), or passthrough (defer to Claude's default permission system)
- [ ] Hook timeout must be short enough to not noticeably delay interactive use (under 5 seconds)

### R2: Fast-Path Classification
Static allowlist/blocklist that resolves common commands without calling Codex.
- [ ] Allowlist: commands that are always safe (read-only operations, standard build commands, git status/log/diff, ls, cat, head, tail, test runners)
- [ ] Blocklist: commands that are always dangerous (rm -rf with broad paths, force pushes to main, DROP/TRUNCATE database operations, curl piped to shell, chmod 777)
- [ ] Classification based on the command's base executable and flags, not string matching on arguments
- [ ] Allowlist and blocklist are user-extensible via configuration
- [ ] When a fast-path match is found, skip Codex entirely

### R3: Codex Safety Classification
Send ambiguous commands to Codex (gpt-5.4-mini) for safety classification.
- [ ] Send the command string plus working directory context to Codex for classification
- [ ] Codex returns a structured verdict: safe (bool), reason (string), severity (info/warn/block)
- [ ] `info` severity: approve silently
- [ ] `warn` severity: approve but log the warning to stderr
- [ ] `block` severity: block execution, return reason to Claude
- [ ] Model configurable via settings (default: gpt-5.4-mini)

### R4: Claude Permission Integration
Integrate with Claude Code's existing allow/block permission system rather than replacing it.
- [ ] Commands already explicitly allowed in Claude's settings (e.g., `Bash(go test *)`) bypass the gate entirely
- [ ] Commands already explicitly blocked in Claude's settings are blocked before the gate runs
- [ ] The gate only evaluates commands that would otherwise prompt the user for permission
- [ ] Gate verdicts respect the same format Claude's permission system uses

### R5: Pattern-Based Verdict Cache
Cache Codex verdicts by normalized command pattern to avoid redundant API calls within a session.
- [ ] Normalize commands by stripping variable arguments (file paths, commit messages, branch names) while preserving the command structure and flags
- [ ] Cache keyed on normalized pattern, scoped to the current session
- [ ] Cache hit returns the stored verdict without calling Codex
- [ ] Cache is in-memory only — does not persist across sessions
- [ ] Cache can be cleared via a command or setting

### R6: Graceful Degradation
When Codex is unavailable, fall back to static rules only.
- [ ] If Codex binary is not installed, use fast-path classification only (R2) and passthrough for everything else
- [ ] If Codex call times out or errors, fall back to passthrough for that command with a stderr warning
- [ ] Never block a command solely because Codex is unreachable

### R7: Configuration
User settings for the command gate.
- [ ] Setting `command_gate` with values `"all" | "interactive" | "off"` — controls which sessions the gate applies to
- [ ] Setting `command_gate_model` to override the Codex model (default: gpt-5.4-mini)
- [ ] Setting `command_gate_timeout` for Codex call timeout in milliseconds (default: 3000)
- [ ] Custom allowlist/blocklist entries additive to the built-in defaults
- [ ] Settings stored alongside other Codex integration config

## Out of Scope
- Reviewing non-Bash tool calls (Edit, Write, etc.)
- Modifying Claude Code's core permission system
- Persistent cross-session learning from past verdicts
- Reviewing commands after execution (post-hoc audit)

## Cross-References
- See also: cavekit-codex-bridge.md (R1 for Codex detection, R2 for shared config mechanism)

## Changelog
- 2026-03-31: Initial draft
