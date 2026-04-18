---
created: "2026-04-18T00:00:00Z"
last_edited: "2026-04-18T00:00:00Z"
---

# Spec: Cavekit Teams — CLI Surface and Integration

## Scope
The user-facing command surface for Cavekit Teams: the `cavekit team` Go subcommand tree, the corresponding `/ck:team` slash commands, the git-hygiene patches (`.gitignore`, `.gitattributes`) that `team init` applies, and the integration points in `scripts/setup-build.sh` and the make-loop dispatch path that enforce team-mode invariants.

## Requirements

### R1: `cavekit team` Subcommand Routing
**Description:** A single `team` subcommand on the existing `cavekit` Go binary that routes to `init`, `join`, `status`, `claim`, `release`, `sync`, and (internal) `heartbeat`.
**Acceptance Criteria:**
- [ ] `cavekit team` with no subcommand prints usage to stderr listing all public subcommands (not `heartbeat`) and exits with code 2
- [ ] Public subcommands: `init`, `join`, `status`, `claim`, `release`, `sync`
- [ ] Internal subcommand: `heartbeat` (used only by the make loop; present in `cavekit team --help` only when `CAVEKIT_INTERNAL=1` is set)
- [ ] Unknown subcommands exit with code 2 and a message `unknown team subcommand: <name>`
- [ ] `cavekit team --help` prints a one-line description per subcommand; exit code 0
- [ ] Every subcommand supports `-h`/`--help` producing its own flag summary; exit code 0
- [ ] Routing lives under `cmd/cavekit/` alongside existing subcommands; no new top-level binary is introduced

**Dependencies:** none

### R2: Subcommand Flag Surface, Exit Codes, and JSON Output
**Description:** Each subcommand has a defined flag set, deterministic exit codes, and an opt-in `--json` mode that prints a single JSON object to stdout for machine consumption (matching Beads' convention).
**Acceptance Criteria:**
- [ ] Every public subcommand accepts `--json`; when set, stdout is a single JSON object followed by `\n` and nothing else; human-readable text is suppressed on stdout (diagnostics still allowed on stderr)
- [ ] `init` flags: `--force` (recreate `.cavekit/team/` scaffolding without touching roster), `--email <addr>`, `--name <str>`; exit codes: 0 ok, 1 already initialized (without `--force`), 2 identity resolution failed (per kit-team R4)
- [ ] `join` flags: `--email <addr>`, `--name <str>`; exit codes: 0 ok, 1 team not initialized, 2 identity resolution failed, 3 already joined (idempotent no-op, still exit 0 unless `--strict`)
- [ ] `status` flags: `--json`, `--task <T-ID>` (restrict to one task), `--user <email>` (restrict to one teammate); exit code 0 always when team is initialized, 1 otherwise
- [ ] `claim` flags: `<task-id>` positional (required), `--json`; exit codes: 0 claimed (or already held), 3 claimed by another user, 4 locally leased by another session, 5 lost claim race on push, 6 task not in frontier
- [ ] `release` flags: `<task-id>` positional (required), `--json`, `--note <str>`; exit codes per kit-team R6
- [ ] `sync` flags: `--json`, `--timeout <seconds>` (default 10); performs `git fetch` + re-read ledger; exit codes: 0 ok, 7 fetch failed (offline)
- [ ] `heartbeat` flags: `<task-id>` positional, `--interval <seconds>` (overrides config), `--once` (single tick then exit); when `--once` absent, runs as daemon until SIGTERM/SIGINT
- [ ] JSON object shapes are versioned with a top-level `schema: "cavekit.team.v1"` key so future changes are detectable

**Dependencies:** R1, kit-team.md

### R3: Slash Commands in `commands/ck/`
**Description:** Slash-command wrappers that shell out to `cavekit team` and are discoverable under the `/ck:team` namespace.
**Acceptance Criteria:**
- [ ] A single file at `commands/ck/team.md` documents all `/ck:team <subcommand>` invocations and their arguments (one file, not seven — reduces slash-command sprawl and matches the pattern of other multi-verb commands in `commands/ck/`)
- [ ] The slash command routes the first positional arg to `cavekit team <subcommand>` with remaining args forwarded verbatim
- [ ] The slash command surfaces `cavekit team` exit codes unchanged so upstream flow control works
- [ ] `/ck:team status` without args runs `cavekit team status` with no `--json` (human output)
- [ ] The slash command documents, at the top of the file, that team mode is opt-in and that `/ck:team init` is the first step
- [ ] The existing `/ck:status` command gains a `--team` flag that delegates to `cavekit team status` (see kit-team-tui.md R6); this requirement only mandates that the flag is not silently consumed by `/ck:status` itself

**Dependencies:** R1, R2

### R4: `.gitignore` Patches Applied by `team init`
**Description:** `cavekit team init` must patch `.gitignore` to hide machine-local state while leaving the committed coordination files tracked.
**Acceptance Criteria:**
- [ ] After `init`, `.gitignore` contains a contiguous block delimited by the exact marker lines `# >>> cavekit-team` and `# <<< cavekit-team`
- [ ] The block contains at minimum: `.cavekit/team/leases/` and `.cavekit/team/identity.json`
- [ ] The block does NOT contain `.cavekit/team/ledger.jsonl`, `.cavekit/team/config.json`, or `context/team/roster.md`
- [ ] Re-running `init` does not duplicate the block; existing markers are detected and the block is replaced in place
- [ ] `init --force` rewrites the block even if markers are missing; absent `--force`, rewriting requires the markers to exist
- [ ] `git check-ignore .cavekit/team/identity.json` exits 0 after `init`; `git check-ignore .cavekit/team/ledger.jsonl` exits non-zero

**Dependencies:** R1

### R5: `.gitattributes` Patches Applied by `team init`
**Description:** `cavekit team init` must configure git's union merge driver for the ledger so concurrent appends from different branches merge automatically.
**Acceptance Criteria:**
- [ ] After `init`, `.gitattributes` contains the exact line `.cavekit/team/ledger.jsonl merge=union` inside a block delimited by `# >>> cavekit-team` / `# <<< cavekit-team`
- [ ] `git check-attr merge -- .cavekit/team/ledger.jsonl` reports `merge: union`
- [ ] Re-running `init` does not duplicate the block or the line
- [ ] If `.gitattributes` does not exist, `init` creates it; the resulting file ends with a single trailing newline
- [ ] `init` logs a warning to stderr if it detects an existing conflicting `merge=` setting for the ledger path set outside the team block; it does NOT silently overwrite user-authored attributes outside the markers

**Dependencies:** R1

### R6: `scripts/setup-build.sh` Refuses Un-Joined Users
**Description:** When team mode is initialized but the local user has not yet run `team join`, `setup-build.sh` must refuse to start a build with a clear error, preventing orphan claims and unidentified commits.
**Acceptance Criteria:**
- [ ] `setup-build.sh` detects team mode by the presence of `.cavekit/team/` as a tracked directory (ledger file exists in git index)
- [ ] In team mode, `setup-build.sh` checks for `.cavekit/team/identity.json`; if absent, it exits with code 10 and prints a message instructing the user to run `cavekit team join`
- [ ] When team mode is not initialized, `setup-build.sh` behavior is unchanged — zero overhead, zero new prompts
- [ ] The check runs before any worktree is created or tmux session is started, so failure is cheap and leaves no side effects
- [ ] The exit-10 path is covered by an automated smoke test or the equivalent inspectable artifact: running `setup-build.sh` in a repo with ledger tracked but no `identity.json` produces exit 10 and no new worktree directory

**Dependencies:** R1, R4

### R7: Task-Dispatch Integration (Claim → Dispatch → Release)
**Description:** The make loop (`/ck:make`, `/ck:make-parallel`) must wrap every task-builder dispatch in a claim/heartbeat/release envelope so teammates observe live ownership and stale sessions self-heal.
**Acceptance Criteria:**
- [ ] Before dispatching a task-builder for `T-X`, the loop invokes `cavekit team claim T-X --json` and parses the result; non-zero exit aborts dispatch and the loop advances to the next frontier task
- [ ] On successful claim, the loop starts a background `cavekit team heartbeat T-X` process whose PID is captured
- [ ] On task-builder exit with success, the loop invokes `cavekit team release T-X` with `complete` semantics (via a `--complete` flag on `release`, or equivalent; the exact flag is the implementor's choice but MUST be documented in R2 help output)
- [ ] On task-builder exit with failure, the loop invokes `cavekit team release T-X --note "<reason>"` and does NOT mark the task complete
- [ ] The heartbeat background process is terminated (SIGTERM, 5s grace, then SIGKILL) in all exit paths including SIGINT of the parent loop
- [ ] When team mode is not initialized, the claim/heartbeat/release steps are no-ops and add zero latency to dispatch
- [ ] The existing `.cavekit/.loop.lock` PID-based single-session lock is preserved unchanged; team leases are in addition to, not a replacement for, `.loop.lock`
- [ ] `task-status.json` is still updated by `mark-complete` on task success; the ledger `complete` event and `task-status.json` completion are written in that order, and a failure between the two is reconciled by kit-team R9

**Dependencies:** R1, R2, kit-team.md (R5, R6, R8)

## Out of Scope
- Interactive prompts or TUI for identity setup (flags only)
- Windows-specific shell integration beyond what the existing `cavekit` binary supports
- Shell-completion scripts for the new subcommands (nice-to-have, not required for V1)
- Migration tooling for pre-team-mode ledgers (no such ledgers exist yet)
- Rate-limiting or throttling of `sync` / `git fetch`
- Cross-repo identity reuse (each checkout runs its own `team join`)

## Cross-References
- See also: kit-team.md (protocol that this CLI surfaces)
- See also: kit-team-tui.md (TUI that reads the same ledger)
- See also: kit-team-overview.md (philosophy and prior art)
- See also: kit-cli.md (existing `cavekit` binary command conventions)
- See also: kit-build-lifecycle.md (make-loop dispatch sequence that R7 wraps)
