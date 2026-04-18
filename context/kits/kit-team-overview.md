---
created: "2026-04-18T00:00:00Z"
last_edited: "2026-04-18T00:00:00Z"
---

# Spec: Cavekit Teams — Overview and Index

## Scope
Index of the three Cavekit Teams kits. Captures the design philosophy every downstream requirement must honor, the dependency graph between the kits, and the prior art we drew from. This document is navigational; it does not add testable requirements on top of its siblings.

## Philosophy

Cavekit Teams is **file-based, git-native, offline-first, grep-able**. Every coordination artifact is a file in the repo or a file under `.cavekit/`. Git is the sync layer between users — not a daemon, not a websocket, not an external service. Leases exist only to prevent races between sessions on the same machine; across machines, append-only union-merged JSONL is the CRDT. A user without network access can still claim, work, and complete; their ledger lines merge cleanly when connectivity returns.

Concretely:

- **File-based**: everything lives in `context/team/` (committed) and `.cavekit/team/` (mixed committed + ignored). No sockets, no ports, no server.
- **Git-native**: the ledger is committed; `merge=union` means concurrent appends never conflict. Identity is `git config user.email`. Sync is `git fetch`.
- **Offline-first**: every claim/release/complete can be made offline and reconciled on next push. Push conflicts rebase and re-verify, never silently overwrite.
- **Grep-able**: ledger is line-oriented JSON. Every claim, heartbeat, completion is visible to `grep`, `jq`, and shell pipelines. No binary formats, no opaque databases.

## Kits in This Package

| Kit | One-line description | Requirements |
|---|---|---|
| `kit-team.md` | Core collaboration protocol: roster, ledger, leases, identity, claim/release/complete, frontier filtering, heartbeats, crash recovery | R1–R9 (9 total) |
| `kit-team-cli.md` | `cavekit team` subcommand surface, slash commands, git-hygiene patches, `setup-build.sh` gate, make-loop dispatch integration | R1–R7 (7 total) |
| `kit-team-tui.md` | Team Activity panel in `cavekit monitor`, fsnotify live updates, git-fetch scheduler, color coding, `/ck:status --team` non-interactive summary | R1–R6 (6 total) |

## Dependency Graph

```
kit-team  ──────────────►  kit-team-cli  ──────────────►  kit-team-tui
(protocol, schemas,          (commands, exit codes,          (TUI panels,
 ledger semantics,            slash wrappers, git              fsnotify watch,
 leases, identity,            hygiene, dispatch                fetch scheduler,
 claim/release/complete,      integration)                     non-interactive
 frontier filter, crash                                        status)
 recovery)
```

- `kit-team-cli` cannot exist without `kit-team` — it is the surface over the protocol.
- `kit-team-tui` cannot exist without both: it reads the ledger defined by `kit-team` and shares a JSON schema with `kit-team-cli`.
- `kit-team` has zero inbound dependencies from the team package; its only external dependencies are existing kits (`kit-site` for frontier, `kit-session` for session lifecycle).

## Prior Art

V1 draws from three open-source multi-agent / multi-user systems. Each line states what we took, not where we diverged.

- **Claude Code Agent Teams** (anthropic/claude-code ecosystem): the "roster as committed markdown" pattern — humans describe themselves and their strengths in a human-readable file, not a config DSL. Our `context/team/roster.md` is this, scoped to a build site.
- **Beads** (steveyegge/beads): the `--json` everywhere + stable schema version discipline. Every `cavekit team` subcommand has a JSON mode with `schema: "cavekit.team.v1"` so scripts and the TUI can consume it without screen-scraping.
- **ccswarm** (nkzw-tech/ccswarm): the "lease + heartbeat, with ledger as system of record" pattern. Machine-local `O_EXCL` locks prevent same-host races; the committed append-only log is the cross-machine truth. We kept that split and left everything else out.

## Cross-References

- kit-team.md
- kit-team-cli.md
- kit-team-tui.md
- kit-site.md — frontier computation that kit-team R7 filters
- kit-session.md — session lifecycle that owns heartbeat processes
- kit-tui.md — host TUI whose theme and conventions kit-team-tui extends
- kit-build-lifecycle.md — make loop whose dispatch kit-team-cli R7 wraps
- kit-cli.md — existing `cavekit` binary conventions that `cavekit team` inherits

## Out of Scope (V1, Across the Entire Team Package)

Restated here so no single kit has to carry the full list:

- Web dashboard or hosted UI
- Real-time websockets, pubsub, or IPC beyond files + git
- Cross-repository teams (one team per repo checkout)
- Per-user permissions, roles, or ACLs
- In-band messaging or mailbox between teammates
- PR-based review workflow or merge gating (future feature)
- Interactive claim/release from inside the TUI (read-only V1)
- Automatic conflict resolution on the roster beyond git's default three-way merge
