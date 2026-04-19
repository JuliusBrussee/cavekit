# Cavekit OpenCode Port

Use the OpenCode Cavekit commands from `~/.config/opencode/commands/` when the repo follows the Cavekit workflow.

Hard rules:
- This is the **Portable Phase 1 OpenCode port**, not upstream Claude/Codex runtime parity.
- Do not claim autonomous loops, stop hooks, worktree fan-out, team mode, or hidden background orchestration.
- Prefer the normal workflow: kits → build-site → scoped implementation → read-only verification.
- Keep requirements implementation-agnostic in kits. Track execution in plans and impl docs.
