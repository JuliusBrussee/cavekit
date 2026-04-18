---
name: ck-team
description: "Team coordination for Cavekit. Opt-in: run `/ck:team init` first, then `/ck:team join` in each checkout."
argument-hint: "<init|join|status|claim|release|sync|next|guard-commit> [args...]"
allowed-tools: ["Bash(cavekit team:*)"]
---

**What this does:** Routes `/ck:team <subcommand>` to the `cavekit team` binary subcommand tree.
**When to use it:** When multiple people or sessions are coordinating work on one build site and you need claims, heartbeats, path-level conflict prevention, or a shared activity log.

## Architecture at a glance

Cavekit's team layer is a **CAS-backed ledger on its own git ref**, plus a
**pre-commit guard** that enforces the ledger on the working tree:

- Ledger lives on `refs/heads/cavekit/team` (orphan branch). Events never touch
  the working branch — your feature-branch diff is never polluted by claims
  or heartbeats.
- Appends go through `--force-with-lease`. A lost race is detected in one
  round trip, not silently merged.
- Claims are **path-scoped**: two people can own disjoint subsystems of the
  same task without blocking each other.
- A pre-commit hook rejects commits that touch files claimed by a teammate.
  Emergency override: `CAVEKIT_TEAM_OVERRIDE=1 git commit ...` (recorded in
  the ledger).
- Offline pushes queue to `.cavekit/team/outbox.jsonl` and drain on the next
  successful op. Status surfaces them with `--conflicts`.

## Important

Team mode is opt-in. The first step is always:

```bash
cavekit team init
```

This also creates the ledger ref on origin and installs
`.git/hooks/pre-commit`.

Then each checkout or clone joins separately:

```bash
cavekit team join
```

## Core commands

```bash
cavekit team status                            # active claims, recent activity, idle members
cavekit team status --conflicts                # add race/override/outbox diagnostics
cavekit team next --json                       # pick the best unclaimed, path-safe task for you
cavekit team claim T-013 --paths "src/auth/**" # claim with file scope
cavekit team release T-013 --complete          # close out ownership
cavekit team sync --timeout 5                  # fetch ledger ref + drain outbox
```

## Configuration (`.cavekit/team/config.json`)

| Field | Default | Meaning |
| --- | --- | --- |
| `lease_ttl_seconds` | `1800` (30 min) | How long a lease stays fresh without a heartbeat |
| `heartbeat_interval_seconds` | `60` | How often the internal heartbeat tick runs |
| `heartbeat_publish_every` | `3` | Publish the batch every N ticks so remote devices see liveness |
| `allow_offline` | `false` | When `false`, claims/releases fail hard if push can't reach origin |

## Dispatch

Forward the first positional argument as the `cavekit team` subcommand and pass the remaining args through verbatim. Propagate the underlying exit code unchanged. Relevant codes:

| Exit | Meaning |
| --- | --- |
| 3 | Task is claimed by another user (or path conflict) |
| 4 | Local lease still fresh in another session |
| 5 | CAS lost on ledger ref — retry after `team sync` |
| 6 | Task not in current frontier |
| 7 | Remote required but push failed (set `allow_offline=true` to bypass) |
| 8 | Pre-commit guard blocked the commit (set `CAVEKIT_TEAM_OVERRIDE=1` to override) |
