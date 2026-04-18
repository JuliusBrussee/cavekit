---
name: ck-team
description: "Team coordination for Cavekit. Opt-in: run `/ck:team init` first, then `/ck:team join` in each checkout."
argument-hint: "<init|join|status|claim|release|sync> [args...]"
allowed-tools: ["Bash(cavekit team:*)"]
---

**What this does:** Routes `/ck:team <subcommand>` to the `cavekit team` binary subcommand tree.
**When to use it:** When multiple people or sessions are coordinating work on one build site and you need claims, heartbeats, or a shared activity log.

## Important

Team mode is opt-in. The first step is always:

```bash
cavekit team init
```

Then each checkout or clone joins separately:

```bash
cavekit team join
```

## Dispatch

Forward the first positional argument as the `cavekit team` subcommand and pass the remaining args through verbatim.

Examples:

```bash
cavekit team status
cavekit team claim T-013
cavekit team release T-013 --complete
cavekit team sync --timeout 5
```

Propagate the underlying `cavekit team` exit code unchanged.
