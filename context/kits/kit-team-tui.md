---
created: "2026-04-18T00:00:00Z"
last_edited: "2026-04-18T00:00:00Z"
---

# Spec: Cavekit Teams — TUI and Status Integration

## Scope
The live-team-activity surface. Adds a "Team Activity" panel to the existing `cavekit monitor` bubbletea TUI, wires filesystem-change notification for the ledger with a polling fallback, schedules periodic `git fetch` to pull in remote activity, and exposes a non-interactive summary via `/ck:status --team`. Purely read-only: the TUI never mutates ledger or lease state.

## Requirements

### R1: Team Activity Panel
**Description:** A dedicated panel in `cavekit monitor` that renders the most recent ledger events in a compact, scannable table.
**Acceptance Criteria:**
- [ ] Panel shows the 20 most recent ledger events ordered newest-first
- [ ] Each row renders exactly four columns: relative time (e.g. `2m ago`), owner (short form — local-part of email when longer than 16 chars), event type (`claim`/`release`/`complete`/`heartbeat`/`note`), task ID (or `—` for team-wide notes)
- [ ] Panel has a visible header "Team Activity (last 20)" and a footer line showing `events: <n>  users: <m>  active: <k>` where `n` is lines in the ledger, `m` is distinct owners seen today, `k` is currently-held claims per kit-team R5
- [ ] Panel is only shown when team mode is initialized (`.cavekit/team/ledger.jsonl` exists); otherwise it is hidden without reserving screen space
- [ ] Panel is scrollable with `j`/`k` (or arrow keys) to page through older events up to 200 lines of history; beyond 200 a footer reads `… older events truncated`
- [ ] Panel rendering is pure function of ledger contents; two TUI processes pointed at the same ledger produce identical visible rows at the same tick

**Dependencies:** kit-team.md R2

### R2: Per-User Active-Claim Summary
**Description:** A secondary strip in the monitor that shows, per teammate, what task they currently hold.
**Acceptance Criteria:**
- [ ] One row per owner who either has a currently-held claim OR appears in `context/team/roster.md` as a team member
- [ ] Each row: owner short form, current task ID (or `idle`), time-since-last-heartbeat (or `—` when idle)
- [ ] Rows with a stale lease (kit-team R3 "stale") are marked with a distinct visual indicator and the tag `(stale)` appended to the task ID
- [ ] Rows are sorted: active claims first (by oldest `acquired_at`), then idle roster members alphabetically by email
- [ ] The current user's own row is always rendered first regardless of sort order when they have an active claim
- [ ] Summary refreshes on the same tick as R1 and shares ledger reads (no double-parsing)

**Dependencies:** R1, kit-team.md R1, kit-team.md R5

### R3: Live Update via Filesystem Notification with Polling Fallback
**Description:** Ledger changes appear in the TUI within one event loop tick of the file being written, whether by the local process or by a `git pull` that fast-forwarded the branch.
**Acceptance Criteria:**
- [ ] `go.mod` adds the dependency `github.com/fsnotify/fsnotify` at a pinned version; `go mod tidy` leaves the module graph clean
- [ ] The monitor registers a watch on `.cavekit/team/ledger.jsonl` and on its parent directory (to survive atomic rename-replace during compaction per kit-team R8)
- [ ] On `WRITE`, `CREATE`, or `RENAME` events, the ledger is re-read and the panel redrawn within one bubbletea tick
- [ ] If fsnotify initialization fails (e.g. inotify watch limit exceeded, unsupported filesystem), the monitor logs a single-line warning and falls back to polling the ledger's mtime every 2 seconds
- [ ] Polling fallback is observable: when `CAVEKIT_TEAM_FORCE_POLL=1` is set, fsnotify is skipped entirely and polling is used; the TUI footer shows `(polling)` in this mode
- [ ] Re-read on each event is bounded: only lines appended since the last byte offset are parsed; full re-read happens only on `CREATE` / `RENAME` (compaction)

**Dependencies:** R1

### R4: `git fetch` Scheduler
**Description:** Periodic background `git fetch` so events from other users land in the local ledger without explicit `cavekit team sync`.
**Acceptance Criteria:**
- [ ] Default fetch interval: 30 seconds; configurable via `.cavekit/team/config.json` key `fetch_interval_seconds`; a value of `0` disables the scheduler
- [ ] Fetch runs in a goroutine and never blocks the UI thread; stdout/stderr of the fetch are discarded, exit codes logged to the monitor's log buffer
- [ ] A fetch that exceeds 10 seconds is cancelled (context timeout); a cancelled fetch is retried at the next interval, not immediately
- [ ] When a fetch fast-forwards the branch and the ledger gains new lines, R3's fsnotify path fires and the panel updates — no separate plumbing
- [ ] The scheduler is paused while the user has unpushed ledger commits on the current branch (detected via `git status --porcelain=v2 --branch`) to avoid masking push-retry state
- [ ] Status-bar indicator shows the last successful fetch time (e.g. `fetch: 12s ago`) and flips to `fetch: offline` after three consecutive failures

**Dependencies:** R3

### R5: Color Coding
**Description:** Visual distinction between own claims, teammate claims, and stale leases that works in both light and dark terminal themes and degrades gracefully without color.
**Acceptance Criteria:**
- [ ] Own claims: one designated accent color (referencing the monitor's existing accent palette — NO new color tokens introduced)
- [ ] Teammate claims: a neutral color distinct from the accent
- [ ] Stale leases: a warning color shared with other warning states in the monitor (consistent reuse, not a new semantic)
- [ ] `complete` events render with a success color; `release` events render in a muted color; `note` events render in the default foreground
- [ ] When `NO_COLOR=1` is set (standard env convention), color output is suppressed and distinction is conveyed by text tags: `[own]`, `[stale]`, `[done]`
- [ ] Every color used in the team panels MUST come from the monitor's existing theme; introducing a new palette entry is a spec violation and a test rejects it by grepping for new color literals in the team-panel source

**Dependencies:** R1, R2, kit-tui.md

### R6: `/ck:status --team` Non-Interactive Summary
**Description:** A printable, pipe-friendly summary of team state that shows the same information as the TUI panels but to stdout, for use in scripts, CI, or a terminal without a TTY.
**Acceptance Criteria:**
- [ ] `/ck:status --team` prints a three-section report: (1) "Active Claims" table (owner, task, acquired-at, stale?), (2) "Recent Activity" table (last 10 events, same columns as R1), (3) "Idle Members" (roster entries with no active claim)
- [ ] `/ck:status --team --json` prints a single JSON object with the same data and the `schema: "cavekit.team.v1"` key (shared with kit-team-cli.md R2); exit code 0 on success, 1 when team mode is not initialized
- [ ] Output is deterministic: given a fixed ledger snapshot and roster, the command produces byte-identical stdout across runs (used for golden tests)
- [ ] The command does NOT mutate state: running it 100 times appends zero new lines to the ledger and creates zero lease files
- [ ] The command completes in under 1 second on a ledger of 10,000 lines (bounded by R3's append-only read optimization)
- [ ] When run without a TTY, color is suppressed regardless of `NO_COLOR` (auto-detection); tables use plain ASCII column separators

**Dependencies:** R1, R2, kit-team-cli.md R2, kit-team-cli.md R3

## Out of Scope
- Interactive claim/release from within the TUI (read-only by design in V1)
- Graphs, charts, or trend visualization
- Desktop notifications or sound on teammate events
- Windowing/multi-panel layout changes beyond the new strip and panel
- User avatars or profile images
- Integration with external presence systems (Slack status, calendar, etc.)

## Cross-References
- See also: kit-team.md (protocol and ledger schema the TUI consumes)
- See also: kit-team-cli.md (commands that produce the ledger events rendered here; shared JSON schema)
- See also: kit-team-overview.md (philosophy and prior art)
- See also: kit-tui.md (host TUI conventions, theme tokens reused by R5)
