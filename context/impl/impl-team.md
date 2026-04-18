---
created: "2026-04-18T00:00:00Z"
last_edited: "2026-04-18T00:00:00Z"
---

# Implementation Tracking: Team Coordination

Build site: context/plans/build-site.md

| Task | Status | Notes |
|------|--------|-------|
| T-001 | DONE | Identity resolution uses `--email`, repo `git config user.email`, then global `git config --global user.email`; persisted to `.cavekit/team/identity.json`. |
| T-002 | DONE | Append-only JSONL ledger with malformed-line skip, stable ordering, and active-claim derivation in `internal/team/ledger.go`. |
| T-003 | DONE | Machine-local lease lockfiles with atomic create, freshness checks, and config-driven TTL in `internal/team/lease.go`. |
| T-004 | DONE | Marker-based `.gitignore` patcher added in `internal/team/gitpatch.go`. |
| T-005 | DONE | Marker-based `.gitattributes` patcher with conflict warning and `merge=union` ledger entry. |
| T-006 | DONE | `context/team/roster.md` template writer and advisory roster parser in `internal/team/roster.go`. |
| T-007 | DONE | `cavekit team` subcommand router added in `cmd/cavekit/team.go`. |
| T-008 | DONE | Per-subcommand flag parsing, exit-code mapping, and `schema: cavekit.team.v1` JSON envelopes. |
| T-009 | DONE | `team init` scaffolds `.cavekit/team/`, writes config/identity, patches git files, and creates roster template. |
| T-010 | DONE | `team join` added with idempotent re-join behavior and `--strict`. |
| T-011 | DONE | `/ck:team` wrapper command added in `commands/team.md`; `/ck:status --team` delegates to `cavekit team status`. |
| T-012 | DONE | Currently-held-claims computation shared across status/claim logic via `internal/team/ledger.go`. |
| T-013 | DONE | `team claim` implements frontier check, lease acquisition, ledger append, commit, and offline-safe push handling. |
| T-014 | DONE | `team release` and `team release --complete` append ledger events, remove lease files, and commit state. |
| T-015 | DONE | Internal `team heartbeat` loop refreshes leases, appends heartbeat events, and auto-releases after repeated failures. |
| T-016 | DONE | `scripts/setup-build.sh` now exits 10 before work begins when team mode is tracked but `identity.json` is missing. |
| T-017 | DONE | `team sync` performs timed `git fetch` and re-reads the ledger. |
| T-018 | DONE | Shared frontier filter added to `internal/site/frontier.go` for excluding active team claims. |
| T-019 | DONE | `team status` human/JSON report includes `frontier_raw`, `frontier_filtered`, and `excluded_by_team`. |
| T-020 | DONE | `/ck:make` and `/ck:make-parallel` instructions now wrap task execution with team claim, heartbeat, and release/complete steps. |
