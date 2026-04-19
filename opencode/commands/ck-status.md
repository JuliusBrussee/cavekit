---
description: Snapshot progress report from kits, plans, impl docs, and git for the OpenCode Cavekit portable workflow
argument-hint: [--filter PATTERN]
---

You are running **/ck-status** for the OpenCode Cavekit portable port.

Hard rules:
- Read-only only.
- No autonomous runtime claims.
- `--watch` is unsupported in this port. If requested, say so plainly and return one snapshot anyway.

Inspect:
- project root
- `AGENTS.md` if present
- `context/kits/cavekit-overview.md` if present
- all relevant `context/kits/cavekit-*.md`
- `context/plans/build-site.md` if present
- `context/impl/` contents if present
- current git status

Return:

## Repo
- project name
- short description

## Kits
- count of kit files
- key domains
- whether kits look draft / active / stale

## Plan
- whether `context/plans/build-site.md` exists
- if present: total tasks, tiers, and current READY frontier
- if absent: say `/ck-map` is next

## Implementation Tracking
- whether `context/impl/` contains active tracking docs
- summarize recent execution notes if present

## Git
- concise changed/untracked summary

## Risks / Gaps
- top 3 blockers or missing artifacts

## Recommended next step
- recommend exactly one of: `/ck-sketch`, `/ck-map`, `/ck-make <target>`, `/ck-check`

Keep it concise and operational.
