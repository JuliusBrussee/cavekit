---
name: ck-check
description: "Inspect the last loop: gap analysis against kits + peer review code review for bugs, security, and quality"
argument-hint: "[--filter PATTERN]"
---

> **Note:** `/bp:inspect`, `/ck:inspect`, `/bp:check` are deprecated aliases. Use `/ck:check` instead.

# Cavekit Inspect — Post-Loop Analysis

Run this after `/ck:make` completes (or is stopped). It does two things:

1. **Gap analysis** — compares what was built against what the kits require
2. **Peer review** — finds bugs, security issues, and quality problems in the code that was written

## Step 0: Resolve Execution Profile

Before starting inspection:

1. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" summary` and print that exact line once.
2. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" model reasoning` and treat the result as `REASONING_MODEL`.
3. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" caveman-active inspect` and treat the result as `CAVEMAN_ACTIVE` (true/false).

Use `REASONING_MODEL` explicitly for the delegated surveyor and inspector work below.

If `CAVEMAN_ACTIVE` is `true`, your own output (status updates, summaries, reasoning) should use caveman-speak: drop articles, filler, pleasantries — keep technical terms exact and code blocks unchanged. The structured report tables (coverage matrix, findings with P0/P1/P2/P3) stay in normal format. Inject `CAVEMAN MODE: ON` into ck:surveyor and ck:inspector subagent prompts so their status reports are also compressed.

## Step 1: Gather Context

Read these files to understand what happened:

1. **Site/Plan** — find in `context/plans/` or `context/sites/` (match `*site*`, `*plan*`, or `*frontier*`, exclude `*overview*`; apply `--filter` from `$ARGUMENTS` if set)
2. **Kits** — all `context/kits/cavekit-*.md` files (apply filter)
3. **Impl tracking** — all `context/impl/impl-*.md` files
4. **Loop log** — `context/impl/loop-log.md`
5. **Git history** — run `git log --oneline -30` to see recent commits from the loop
6. **Git diff** — run `git diff main...HEAD` (or appropriate base branch) to see all code changes
7. **Design system** — read `DESIGN.md` at project root if it exists (for design compliance checking in Steps 2-3)

If no impl tracking or loop log exists, tell the user:
> No loop artifacts found. Run `/ck:make` first, then `/ck:check` after it completes.

## Step 2: Gap Analysis

Do not perform the substantive gap analysis inline. Dispatch a `ck:surveyor` subagent with `model: "{REASONING_MODEL}"` to produce the coverage analysis.

For every cavekit requirement (R-numbered) and its acceptance criteria, determine status:

| Status | Meaning |
|--------|---------|
| **COMPLETE** | All acceptance criteria met, code exists, tests pass |
| **PARTIAL** | Some criteria met, others missing or untested |
| **MISSING** | Not implemented at all |
| **OVER-BUILT** | Code exists that no cavekit requires |

**How to verify:**
- Don't trust impl tracking alone — check the actual code
- For each acceptance criterion, find the code that satisfies it
- Check if tests exist and whether they actually test the criterion
- Run the test suite if possible: `npm test`, `pytest`, `cargo test`, etc.

**Design system compliance (if DESIGN.md exists):**
For every UI-related acceptance criterion, also check:
- Implemented colors match the DESIGN.md palette (Section 2)
- Typography follows the DESIGN.md type scale (Section 3)
- Component styling matches DESIGN.md patterns (Section 4)
- Spacing follows the DESIGN.md layout principles (Section 5)
- Responsive behavior matches DESIGN.md breakpoints (Section 8)

Report violations as status `DESIGN VIOLATION` — implementation exists but deviates from the design system.

## Step 3: Peer Review Code Review

Do not perform the substantive code review inline. Dispatch a `ck:inspector` subagent with `model: "{REASONING_MODEL}"` to review the loop diff and completed tasks.

Review all code changes from the loop (`git diff` output) looking for:

**Bugs**
- Logic errors, off-by-one, null/undefined handling
- Race conditions, deadlocks
- Edge cases not covered
- Error handling that swallows failures silently

**Security**
- Input validation gaps (injection, XSS, path traversal)
- Auth/authz bypasses
- Data exposure in logs, errors, or API responses
- Hardcoded secrets

**Performance**
- O(n^2+) on unbounded data
- Missing pagination
- N+1 queries
- Synchronous blocking where async is needed
- Resource leaks

**Quality**
- Dead code, unused imports
- Unnecessary complexity or abstraction
- Inconsistency with existing codebase patterns
- Missing error handling on external calls

**Cavekit Gaps**
- Requirements that SHOULD exist but don't
- Edge cases the cavekit doesn't address
- Integration points between domains that are undefined

**Design System Violations** (if DESIGN.md exists)
- Hardcoded color values that should use design tokens
- Typography that doesn't follow the defined type scale
- Component styling that deviates from DESIGN.md patterns
- Spacing/layout that doesn't follow the spacing scale
- Missing responsive behavior defined in DESIGN.md

## Step 4: Generate Report

Present this to the user:

```markdown
# Cavekit Inspect Report

**Date:** {date}
**Loop iterations:** {from loop-log.md}
**Commits:** {count from git log}
**Files changed:** {count}

---

## Gap Analysis

### Coverage
| Status | Requirements | Acceptance Criteria | % |
|--------|-------------|--------------------|----|
| COMPLETE | {n} | {n} | {%} |
| PARTIAL | {n} | {n} | {%} |
| MISSING | {n} | {n} | {%} |
| OVER-BUILT | {n} | — | — |

### Progress Bar
[████████████████░░░░] 82% coverage

### Gaps Found

#### PARTIAL — needs more work
| Requirement | Cavekit | What's Done | What's Missing |
|------------|------|-------------|----------------|
| R{n}: {name} | cavekit-{domain}.md | {met criteria} | {unmet criteria} |

#### MISSING — not started
| Requirement | Cavekit | Why |
|------------|------|-----|
| R{n}: {name} | cavekit-{domain}.md | {not in site / blocked / dead end} |

#### OVER-BUILT — no cavekit for this
| Feature | Files | Recommendation |
|---------|-------|---------------|
| {feature} | {files} | Add cavekit / Remove code |

---

## Peer Review

### Findings: {total} ({P0 count} critical, {P1 count} high, {P2 count} medium, {P3 count} low)

#### P0 — Critical (blocks release)
**F-001: {title}**
- File: {path}:{lines}
- Issue: {what's wrong}
- Evidence: {code snippet or test result}
- Fix: {specific action}

#### P1 — High (should fix before merge)
...

#### P2 — Medium
...

#### P3 — Low
...

---

## Design System Compliance (if DESIGN.md exists)

### Violations: {count}
| Component | File | Violation | DESIGN.md Reference | Severity |
|-----------|------|-----------|---------------------|----------|
| {component} | {path} | {what deviates} | Section {N} | P2/P3 |

---

## Verdict

**{APPROVE / REVISE / REJECT}**

- APPROVE: No P0/P1 findings, coverage > 90%
- REVISE: P1 findings or coverage 70-90%
- REJECT: P0 findings or coverage < 70%

## Recommended Next Steps
1. {highest priority action}
2. {next action}
3. {if gaps exist: run `/ck:make` again to address remaining work}
4. {if cavekit gaps found: kits will be updated below, then `/ck:map` + `/ck:make`}
```

## Step 5: Revise

After presenting the report, **automatically update kits and site** based on findings. Do not ask — just do it.

### Update Kits

For each finding that reveals a cavekit gap:

- **Missing requirement** — add it to the appropriate cavekit file as a new R-number with acceptance criteria
- **Ambiguous criterion** — rewrite the criterion to be specific and testable
- **Untestable criterion** — rewrite to be automatically verifiable, or flag with `[HUMAN REVIEW]`
- **Over-built feature worth keeping** — add a new requirement to formalize it
- **Over-built feature not worth keeping** — note in the report but don't add to cavekit
- **Security/bug finding that exposes a cavekit gap** — add a requirement that would have caught it (e.g. "R7: Input Validation — all user input is sanitized before database queries")
- **Design violation that reveals a missing DESIGN.md pattern** — update DESIGN.md by adding the missing pattern to the appropriate section, and log the change to `context/designs/design-changelog.md`

When modifying a cavekit file:
- Update the `last_edited` date in frontmatter
- Add new requirements at the end of the existing requirements list
- Add a `## Changes` section at the bottom noting what was added and why:

```markdown
## Changes
- {date}: Added R{n} ({title}) — discovered during inspection (finding F-{n})
```

### Update Site

If new requirements were added to kits, add corresponding tasks to the site:

1. Read the build site (check `context/plans/` first, then `context/sites/`)
2. For each new requirement, create task(s) with T-numbers continuing from the last existing task
3. Place tasks in the appropriate tier based on dependencies
4. Update the `last_edited` date in frontmatter
5. Update the summary table at the bottom

### Restore Archived Impl Tracking

If the verdict is **REVISE** or **REJECT** and `context/impl/` has no `impl-*.md` files (they were archived by a previous `/ck:make` run), restore them so the next build cycle knows which tasks are already done:

1. Find the most recent archive: `ls -td context/impl/archive/*/ 2>/dev/null | head -1`
2. If found, copy all `impl-*.md` files (NOT `loop-log.md` or `peer-review-findings.md`) back to `context/impl/`:
   ```bash
   LATEST_ARCHIVE=$(ls -td context/impl/archive/*/ 2>/dev/null | head -1)
   if [[ -n "$LATEST_ARCHIVE" ]]; then
     for f in "$LATEST_ARCHIVE"/impl-*.md; do
       [[ -f "$f" ]] && cp "$f" context/impl/
     done
     echo "♻️  Restored impl tracking from $LATEST_ARCHIVE"
   fi
   ```
3. Report the restoration to the user

This ensures the next `/ck:make` cycle can compute the correct frontier (skipping already-done tasks).

### Update Impl Tracking

For each peer review finding (bugs, security, performance):

1. Read or create `context/impl/impl-review-findings.md`
2. Log each finding with status NEW:

```markdown
---
created: "{CURRENT_DATE_UTC}"
last_edited: "{CURRENT_DATE_UTC}"
---

# Review Findings

| Finding | Severity | File | Status |
|---------|----------|------|--------|
| F-001: {title} | P0 | {path} | NEW |
| F-002: {title} | P1 | {path} | NEW |
```

These findings will be picked up by the next `/ck:make` loop — the build prompt reads impl tracking and prioritizes P0 issues first.

### Report What Changed

After revision, tell the user:

```markdown
## Revision Summary

### Kits Updated
| Cavekit | Changes |
|------|---------|
| cavekit-{domain}.md | Added R{n}: {title} |

### Site Updated
| Task | Title | Tier | From Finding |
|------|-------|------|-------------|
| T-{n} | {title} | {tier} | F-{n} |

### Findings Logged
{n} findings written to context/impl/impl-review-findings.md

Ready for next cycle: `/ck:make`
```
