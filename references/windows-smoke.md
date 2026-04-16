# Windows Smoke Harness

Use [`scripts/windows-smoke.ps1`](../scripts/windows-smoke.ps1) to run the manual Windows acceptance pass for the currently implemented native surface:

- `install.ps1`
- `cavekit monitor`
- the Go-owned `config`, `command-gate`, `codex-review`, `sync-codex`, `debug`, `status`, `kill`, and `reset` subcommands

## Preconditions

- Windows 10 or Windows 11 with ConPTY available
- Git, Go, Claude Code, and `claude` on `PATH`
- A repo clone at `$HOME\\.cavekit` or another writable location
- Baseline pass should **not** have `codex` installed

## Recommended Flow

1. Start from a fresh VM snapshot or a throwaway Windows user profile.
2. Clone the repo and open PowerShell in the repo root.
3. Run the baseline pass first:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\windows-smoke.ps1
```

4. For the optional confidence pass with Codex installed:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\windows-smoke.ps1 -Pass codex-present
```

## Useful Flags

- `-SkipInteractive`: skip the monitor/TUI manual steps and mark them as pending
- `-SkipBuildAndTest`: skip the initial `go build ./...` and `go test ./...`
- `-SkipInstall`: skip `install.ps1` and only validate the current installed binary
- `-OutputDir <path>`: write logs, transcript, screenshots, and markdown report to a custom directory
- `-NoTranscript`: skip `Start-Transcript`

Example:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\windows-smoke.ps1 `
  -SkipInteractive `
  -OutputDir "$HOME\\smoke-runs\\cavekit-win"
```

## What The Script Covers

- Verifies toolchain prerequisites and optional `codex` state
- Runs `go build ./...` and `go test ./...` unless skipped
- Runs `install.ps1`, then validates:
  - `%LOCALAPPDATA%\\Programs\\cavekit\\bin\\cavekit.exe`
  - user `PATH` visibility
  - Claude marketplace/plugin artifacts
  - Codex plugin, marketplace, and prompt sync artifacts
  - reinstall idempotence
- Exercises the new Go-owned subcommands and records their output
- Guides the operator through the Windows monitor flow and then validates:
  - `windowspty-sessions.json`
  - session log creation
  - `status`, `kill`, and `reset`
- Writes a markdown report plus per-command logs

## Manual Monitor Checklist

The repo itself is the monitor fixture. The `n` picker should list sites derived from `context/plans/build-site*.md`.

During the interactive phase:

1. Launch one site and confirm a live instance row appears.
2. Stay on Preview long enough to confirm output refreshes.
3. Press `Enter` on the selected instance and confirm Windows stays in the TUI and switches to the Terminal tab.
4. Press `i`, send a harmless input, press `Enter`, then `Esc`, and confirm new output appears.
5. Launch a second site and confirm `j`/`k` switch between both sessions.
6. Quit with `q` so the script can validate `kill` and `reset`.

## Expected Windows Limitations

These are expected outcomes for the current milestone and should not be marked as blockers:

- `cavekit setup-build` exits non-zero on Windows with the native-porting placeholder message.
- `Open` in the TUI does **not** launch a full-screen external attach session on Windows.
- The Windows Terminal tab shows session scrollback instead of a separate free-shell worktree terminal.

## Outputs

Each run writes:

- a markdown smoke report
- a PowerShell transcript unless `-NoTranscript` is used
- one log file per command invocation
- a screenshots directory for the manual capture set

Use [`references/windows-smoke-report-template.md`](windows-smoke-report-template.md) if you want to hand-author a report alongside the generated one.
