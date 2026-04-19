---
description: Read-only verification against kits and code for the OpenCode Cavekit portable workflow
argument-hint: [--filter PATTERN]
---

You are running **/ck-check** for the OpenCode Cavekit portable port.

Hard rules:
- Read-only analysis only.
- Do not edit kits, plans, code, or impl docs.
- Do not claim upstream inspection/runtime parity.
- Inspect files directly. Do not guess from summaries alone.

Interpret `$ARGUMENTS` as an optional filter for kits, domains, or task ranges.

## Inspect
Use Read/Glob/Grep and, when useful, Bash for git/test commands. Inspect:
- `AGENTS.md` if present
- relevant `context/kits/cavekit-*.md`
- `context/kits/cavekit-overview.md` if present
- `context/plans/build-site.md` if present
- `context/impl/` artifacts if present
- relevant source files
- relevant tests
- `git status`
- recent diff or recent commits when useful

## Pass 1: Gap analysis
For each relevant requirement and acceptance criterion, classify:
- COMPLETE
- PARTIAL
- MISSING
- OVER-BUILT

Ground each finding in actual code or tests with file references.

## Pass 2: Code review
Look for:
- logic bugs
- edge-case misses
- poor error handling
- security issues
- performance issues
- maintainability problems
- drift from repo rules in `AGENTS.md`

## Output
Return a concise report with:
- coverage summary
- top requirement gaps
- prioritized findings (P0/P1/P2/P3)
- verdict: APPROVE / REVISE / REJECT
- exact next step, usually `/ck-make <target>` or `/ck-sketch`

Do not auto-fix anything. Recommend fixes only.
