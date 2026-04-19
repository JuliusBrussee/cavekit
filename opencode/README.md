# Cavekit for OpenCode — Portable Phase 1

This directory contains the **OpenCode-native Cavekit port** for the portable phases only:

- `/ck-init`
- `/ck-sketch`
- `/ck-map`
- `/ck-make`
- `/ck-check`
- `/ck-status`
- `/ck-help`

## What this port is

- Real OpenCode command files, stored in-repo and installable via `install.sh`
- Truthful adaptation of Cavekit's **spec → plan → build → verify** workflow
- Sequential, tool-driven execution using OpenCode's normal file and shell tools

## What this port is not

- Not the Claude Code plugin runtime
- Not the Codex prompt/runtime bundle
- No autonomous stop-hook loop
- No `Agent()` fan-out or worktree orchestration
- No `.cavekit/tasks.json` task registry
- No `make-parallel`, team mode, visual companion, or auto-backprop runtime

## Install into OpenCode

Run:

```bash
~/.cavekit/install.sh
```

If OpenCode is detected, the installer symlinks `opencode/commands/*.md` into:

```text
~/.config/opencode/commands/
```

It also links `opencode/AGENTS.md` **only when** `~/.config/opencode/AGENTS.md` does not already exist.

## Command contract

### `/ck-init`
- Bootstraps `context/` directories and minimal CLAUDE docs
- Optional `--tools-only` capability summary
- Does **not** create the upstream `.cavekit/` runtime

### `/ck-sketch`
- Design-first kit drafting
- Requires explicit approval before writing kit files
- Supports interactive and `--from-code` brownfield drafting

### `/ck-map`
- Reads kits and writes `context/plans/build-site.md`
- Produces tier tables, a coverage matrix, and a Mermaid dependency graph
- Does **not** initialize `.cavekit/tasks.json`

### `/ck-make`
- Sequential, scoped implementation for one task or requirement at a time
- Runs tests/builds with OpenCode's normal shell tool
- Does **not** auto-commit, auto-loop, or spawn subagents

### `/ck-check`
- Read-only verification against kits and code
- Reports gaps and prioritized review findings
- Does **not** auto-fix or auto-revise kits

### `/ck-status`
- Single-snapshot status report from kits, plan, impl docs, and git
- `--watch` is intentionally unsupported in this port

## Verification

Automated check:

```bash
node tests/run-tests.cjs
```

Manual smoke test in a target repo:

1. `/ck-init`
2. `/ck-sketch --from-code`
3. Approve domain proposal and write kits
4. `/ck-map`
5. `/ck-status`
6. `/ck-make T-001` or similar target
7. `/ck-check`

## Port honesty rule

Do not claim parity with upstream Cavekit runtime. Describe this as an **OpenCode portable port**.
