---
created: "2026-04-18T00:00:00Z"
last_edited: "2026-04-18T00:00:00Z"
---

# Spec: Cavekit Teams — Core Collaboration Protocol

## Scope
The file-based, git-native, offline-first coordination protocol that lets multiple users build on a single `context/plans/build-site.md` without stepping on each other's work. Defines the roster, the append-only ledger, machine-local leases, identity resolution, and the atomic claim/release/complete protocol that gates frontier dispatch.

Cavekit Teams extends existing Cavekit state; it does NOT introduce a daemon, websocket, message bus, or external service. Git is the sync layer between users. Leases prevent races between sessions on the same machine.

## Requirements

### R1: Roster File
**Description:** A human-authored markdown file that names the team, their focus areas, and their "personal bests" (domains they ship fastest in). Committed to git; normal three-way merge resolves concurrent edits.
**Acceptance Criteria:**
- [ ] File path: `context/team/roster.md`
- [ ] `cavekit team init` creates the file from a template when absent; never overwrites an existing roster
- [ ] Template contains an H1 title, an H2 "Members" section, and at least one example member block with subheadings `Focus`, `Personal bests`, and `Identity` (the git email used on commits)
- [ ] Roster is NOT listed in `.gitignore`; `git check-ignore context/team/roster.md` exits non-zero after `team init`
- [ ] File has no required schema beyond markdown conventions; parsing failures MUST NOT block the claim protocol — roster is advisory for display, never gating
- [ ] The template documents that member `Identity` values are matched case-insensitively against ledger `owner` fields to link activity to a roster entry

**Dependencies:** none

### R2: Ledger Format
**Description:** An append-only JSON-Lines file that records every claim, release, completion, heartbeat, and note. Committed to git with union-merge so concurrent appends from multiple users never conflict.
**Acceptance Criteria:**
- [ ] File path: `.cavekit/team/ledger.jsonl`
- [ ] Each line is a single JSON object terminated by `\n`; no trailing commas; no array wrapping
- [ ] Required keys on every line: `ts` (RFC3339 UTC, e.g. `2026-04-18T12:34:56Z`), `type` (one of `claim`, `release`, `complete`, `heartbeat`, `note`), `task` (task ID matching the site's `T-*` pattern, or empty string for team-wide notes), `owner` (the identity string from R4), `host` (hostname), `session` (UUID v4)
- [ ] `claim` and `heartbeat` lines additionally carry `lease_until` (RFC3339 UTC, >= `ts`)
- [ ] `note` lines additionally carry `note` (free-form string, newlines escaped as `\n`)
- [ ] Append-only invariant: `cavekit team` subcommands MUST NOT rewrite, reorder, or delete existing lines; only append
- [ ] `.gitattributes` contains the line `.cavekit/team/ledger.jsonl merge=union` after `cavekit team init`; `git check-attr merge .cavekit/team/ledger.jsonl` reports `merge: union`
- [ ] Malformed lines (JSON parse failure) are logged to stderr and skipped; they do NOT abort ledger reads
- [ ] Events are ordered by `ts`; ties broken lexicographically by `session` then line number

**Dependencies:** R4

### R3: Leases (Machine-Local Single-Writer Lock)
**Description:** Per-task lockfiles that serialize claims within a single machine checkout, created atomically so two shells on the same host cannot both claim the same task.
**Acceptance Criteria:**
- [ ] Lease path: `.cavekit/team/leases/T-<ID>.lock`; the `.cavekit/team/leases/` directory is listed in `.gitignore` and `git check-ignore` on any file inside it exits 0
- [ ] Lease creation uses `O_EXCL`-equivalent atomic-create semantics; a second create against an existing lease exits non-zero with a distinguishable error (exit code reserved in R5)
- [ ] Lease contents are JSON with keys `owner`, `host`, `pid`, `session`, `acquired_at` (RFC3339 UTC), `heartbeat_at` (RFC3339 UTC), `expires_at` (RFC3339 UTC)
- [ ] Default TTL from `acquired_at` to `expires_at`: 10 minutes; default heartbeat cadence: 60 seconds
- [ ] A lease is "fresh" iff `now < expires_at` AND `now - heartbeat_at < TTL`
- [ ] A lease is "stale" iff it fails the freshness check; stale leases MAY be stolen per R9 after ledger cross-check
- [ ] TTL and heartbeat cadence are configurable via `.cavekit/team/config.json` keys `lease_ttl_seconds` and `heartbeat_interval_seconds`; defaults apply when absent

**Dependencies:** R2

### R4: Identity Resolution
**Description:** The string that identifies "who" owns a claim. Derived deterministically from git configuration at team-init time and persisted locally so later commands do not re-derive.
**Acceptance Criteria:**
- [ ] Resolution order at `cavekit team init` and `cavekit team join`: (1) `--email` flag if provided, (2) `git config user.email` in the current repo, (3) `git config --global user.email`, (4) hard-fail with exit code 2 and a message instructing the user to set `git config user.email`
- [ ] Resolved identity is written to `.cavekit/team/identity.json` with keys `email`, `name` (optional, from `--name` or `git config user.name`), `session` (UUID generated once), `joined_at` (RFC3339 UTC)
- [ ] `.cavekit/team/identity.json` is listed in `.gitignore`; `git check-ignore` on it exits 0
- [ ] Identity is NEVER inferred from hostname, `$USER`, or environment variables
- [ ] All ledger lines emitted by a given checkout use the `email` value from `identity.json` as their `owner` field; `grep owner .cavekit/team/ledger.jsonl` produces identical `owner` values for all lines authored by one checkout
- [ ] If `identity.json` is missing when a `cavekit team` subcommand other than `init`/`join` runs, the subcommand exits non-zero with a message instructing the user to run `cavekit team join`

**Dependencies:** none

### R5: Claim Protocol
**Description:** The atomic six-step sequence that moves a task from "available in frontier" to "claimed by the current user," with rebase-retry semantics on push conflict.
**Acceptance Criteria:**
- [ ] Step 1: best-effort `git pull --ff-only` on the repo; a non-zero exit (offline, diverged, no upstream) is logged and the claim continues — offline claiming is allowed
- [ ] Step 2: read `.cavekit/team/ledger.jsonl`, compute the set of currently-held claims (a `claim` line whose most recent follow-up for the same `task`+`session` is NOT `release` or `complete`, AND either `lease_until` > now OR a `heartbeat` line for the same `session` exists within lease TTL)
- [ ] Step 3: if the target task is in the currently-held set and its owner is NOT the current identity, the claim fails with exit code 3 ("task claimed by another user") and prints the conflicting owner
- [ ] Step 4: atomically create `.cavekit/team/leases/T-<ID>.lock` (R3); on collision with a fresh lease owned by another session, fail with exit code 4 ("task locally leased"); on collision with a stale lease, proceed per R9
- [ ] Step 5: append one `claim` line to `.cavekit/team/ledger.jsonl` (R2) with `lease_until = now + TTL`
- [ ] Step 6: `git add .cavekit/team/ledger.jsonl && git commit -m "claim: T-<ID>" --allow-empty` as a separate commit; commit SHA is printed on stdout
- [ ] Push conflict recovery: if a subsequent `git push` fails because the remote advanced, the client MUST `git pull --rebase`, re-run Step 2 on the merged ledger, and: if no conflicting `claim` with earlier `ts` landed, retain the claim; if a conflicting earlier claim landed, append a `release` line for the losing claim, delete the lease, and exit non-zero with code 5 ("lost claim race")
- [ ] Claim is idempotent: re-claiming a task the current identity already holds is a no-op with exit code 0 and a single stdout line `already claimed: T-<ID>`

**Dependencies:** R2, R3, R4

### R6: Release and Complete Protocol
**Description:** Voluntary relinquishment (`release`) and successful termination (`complete`) of a claim, both of which remove the lease and record the event.
**Acceptance Criteria:**
- [ ] `release`: append a `release` line to the ledger, delete the matching `.cavekit/team/leases/T-<ID>.lock`, commit the ledger with message `release: T-<ID>`; exit code 0
- [ ] `complete`: append a `complete` line to the ledger, delete the matching lease, commit with message `complete: T-<ID>`; exit code 0
- [ ] Releasing a task not currently claimed by the local identity is a no-op with exit code 0 and a stdout warning; it MUST NOT append a ledger line
- [ ] Completing a task not currently claimed by the local identity exits non-zero with code 6 ("cannot complete unclaimed task")
- [ ] `complete` events are authoritative for team-level "done" state, independent of `task-status.json`; the two are reconciled per R9
- [ ] After release or complete, the task reappears in the frontier for other users on the next ledger read

**Dependencies:** R2, R3, R5

### R7: Frontier Filtering
**Description:** The logic that removes actively-claimed tasks from the set returned by `ReadyTasks`, so parallel users never receive the same task.
**Acceptance Criteria:**
- [ ] Filtering runs AFTER the existing dependency-and-status frontier computation, never before
- [ ] A task is excluded from the filtered frontier iff it appears in the currently-held-claims set (R5 Step 2) with an owner OTHER than the current identity
- [ ] A task claimed by the current identity is excluded from the frontier UNLESS it is the current session's active task; this prevents re-dispatch of one's own in-flight work
- [ ] Filtering is applied identically by `/ck:make`, `/ck:make-parallel`, and `/ck:team status`; a single shared implementation is referenced by all three
- [ ] When team mode is not initialized (no `.cavekit/team/identity.json`), filtering is a pass-through and imposes zero overhead
- [ ] The filtered frontier is observable: `cavekit team status --json` reports `frontier_raw`, `frontier_filtered`, and `excluded_by_team` arrays of task IDs

**Dependencies:** R5

### R8: Heartbeat Behavior During Active Task
**Description:** While a task-builder is running, its owning session MUST keep the lease fresh so teammates see live activity and do not steal the task.
**Acceptance Criteria:**
- [ ] A heartbeat tick writes `heartbeat_at = now` and `expires_at = now + TTL` to the lease file atomically (write-to-temp-and-rename)
- [ ] A heartbeat tick appends a single `heartbeat` ledger line with matching `lease_until`
- [ ] Heartbeat cadence matches R3's configured `heartbeat_interval_seconds` (default 60s); jitter up to +/- 10% is allowed
- [ ] Heartbeat ticks run for the lifetime of the owning task-builder and stop when the claim is released or completed
- [ ] Heartbeat ticks that cannot write the lease file (e.g. disk full, directory removed) log to stderr and attempt one retry; two consecutive failures trigger an auto-release with ledger `release` line containing `note: "heartbeat failure"`
- [ ] Ledger growth from heartbeats is bounded: operators can truncate heartbeat lines older than 24h via `cavekit team compact`; the compaction rewrites the file as a single atomic replace and commits with message `compact: ledger`

**Dependencies:** R2, R3

### R9: Crash Recovery and Stale-Lease Reconciliation
**Description:** When a session crashes (host reboot, `kill -9`, network partition during push), the next user to touch the task must be able to detect the crash and reclaim safely without corrupting history.
**Acceptance Criteria:**
- [ ] Startup reconciliation: `cavekit team` subcommands read the ledger and, for every `claim` line whose `owner` + `session` match the local `identity.json` but whose matching lease file is missing, append a `release` line with `note: "crash recovery"` and commit
- [ ] Stale-lease steal: a claim attempt against a task with a stale lease (R3) MUST verify against the ledger that no fresh `heartbeat` exists for that `session` within TTL; only then may it delete the stale lease, append a `release` line with `note: "stolen stale <owner>"`, and proceed with R5 Step 4
- [ ] Stolen leases are always recorded: `grep '"note":"stolen stale' .cavekit/team/ledger.jsonl` lists every theft, owner included
- [ ] Reconciliation is idempotent: running it twice in succession produces no additional ledger lines the second time
- [ ] A completion event for a task that has no prior matching claim (e.g. legacy `task-status.json` marked it done) is accepted and logged with `note: "backfill"`; subsequent frontier queries treat the task as done
- [ ] Reconciliation never deletes ledger lines; it only appends

**Dependencies:** R2, R3, R5, R6

## Out of Scope
- Web dashboard or hosted UI
- Real-time websocket or pubsub transport between users (git fetch is the only sync mechanism)
- Cross-repository teams (one team per repo checkout)
- Per-user permissions, roles, or ACLs
- In-band messaging or mailbox between teammates
- PR-based review workflow or merge gating (future feature)
- Automatic conflict resolution on the roster file beyond git's default three-way merge

## Cross-References
- See also: kit-team-cli.md (command surface, exit codes, integration hooks)
- See also: kit-team-tui.md (live team activity panel and status flag)
- See also: kit-team-overview.md (philosophy, prior art, dependency graph)
- See also: kit-site.md (frontier computation that R7 filters)
- See also: kit-session.md (session lifecycle that owns heartbeats)
