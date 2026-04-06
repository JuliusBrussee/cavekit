---
name: ck-revise
description: Trace recent manual code fixes back into kits and context files
---

> **Note:** `/bp:revise` is deprecated and will be removed in a future version. Use `/ck:revise` instead.

# Cavekit Revise — Trace Fixes to Kits

You are performing revision: tracing recent manual code changes back into kits, plans, and context files so that the convergence loop can reproduce them autonomously. The core principle: when a fix exists only in code without a corresponding cavekit update, the iteration loop may reintroduce the same defect.

## Step 1: Analyze Recent Commits

Determine the revision range:
1. Read each cavekit's `## Changelog` section for the most recent entry's commit SHAs
2. Use the newest commit SHA across all kits as the "since" marker
3. If no changelog entries exist in any cavekit, fall back to `git log --oneline -20`
4. Run `git log --oneline <since>..HEAD` to gather commits since last revision

Read the diffs for each commit.

Classify each commit into one of three categories:

| Category | Description | Action |
|----------|-------------|--------|
| **Manual fix** | Human-authored code change fixing a bug or adding behavior | Revise (proceed to Step 2) |
| **Iteration loop** | Changes made by an automated convergence loop session | Skip — already cavekit-driven |
| **Infrastructure** | Build config, CI, tooling, dependency updates | Skip — not cavekit-relevant |

Report the classification table to the user.

## Step 2: Analyze Each Manual Fix

For each commit classified as a manual fix, determine:

1. **WHAT** changed — which files, which functions, what behavior was added/modified/removed
2. **WHY** it changed — from the commit message, PR description, diff context, and surrounding code comments
3. **RULE** — what invariant or requirement was violated that necessitated this fix
4. **LAYER** — which cavekit, plan, or prompt should have caught this:
   - **Cavekit gap**: the requirement was never specified
   - **Plan gap**: the cavekit existed but the plan didn't implement it
   - **Prompt gap**: the plan existed but the prompt didn't guide the agent to it
   - **Validation gap**: everything existed but no test caught the regression
5. **VISUAL** — does this fix change visual appearance (CSS, styling, layout, colors, typography)? If yes:
   - Does `DESIGN.md` cover this pattern? If not, this is a **design system gap**
   - If yes but the code deviated, this is a **design compliance issue**

## Step 3: Map Changes to Cavekit Requirements

For each manual fix, identify which cavekit requirements are affected using the CLAUDE.md hierarchy as the primary traversal path:

1. **Check source-tree CLAUDE.md files first** — for each changed file, read the nearest `CLAUDE.md` in its directory (or parent directories). These files contain direct cavekit references like "implements cavekit-auth.md R2". This is the fastest path to the affected requirements.
2. **Read all kits** in `context/kits/` to build a requirement index (cavekit file → R-numbers → descriptions)
3. **Read the build site** in `context/plans/` (or `context/sites/` for legacy projects) to map tasks → requirements → kits
4. **Match changed files** to tasks in the build site (check task titles, impl tracking files, and git blame for task-ID references in commit messages)
5. **Identify affected requirements** by tracing: changed file → CLAUDE.md → cavekit requirement → cavekit

For CSS/styling/UI component changes, also check `DESIGN.md` at project root. If the fix changes colors, typography, spacing, or component appearance, trace to the relevant DESIGN.md section.

For each affected requirement, record:
| Cavekit | Requirement | Current Description | What Changed | Needs Update? |

**Unaffected requirements remain untouched.** Only modify requirements where the code change reveals a gap, ambiguity, or behavioral shift.

**Cross-cavekit moves:** If a fix reveals that a requirement belongs in a different domain:
1. Move the requirement to the correct cavekit (assign the next available R-number)
2. Add a cross-reference in the original cavekit: "R{n} moved to cavekit-{domain}.md R{m}"
3. Update both kits' `## Cross-References` sections

## Step 3b: Discover Governing Plan Files

For each changed source file, determine which plan file governs it:
- First check the directory's `CLAUDE.md` for build task references (e.g., "Build tasks: T-004, T-005")
- Search `context/plans/` (or `context/sites/` for legacy projects) for plan files that reference the changed paths
- If no plan covers the file, flag it as an **untracked file** (potential cavekit gap)

## Step 4: Update Kits and Context Files

For each manual fix, update the appropriate context files:

### Cavekit Updates (context/kits/)
- If the fix reveals a missing requirement, add it with testable acceptance criteria
- If the fix reveals an ambiguous requirement, clarify it
- Update the `last_edited` frontmatter date to today's date
- **Append a changelog entry** to the cavekit's `## Changelog` section:

```markdown
### {YYYY-MM-DD} — Revision
- **Affected:** R{n}, R{m}
- **Summary:** {what changed and why}
- **Commits:** {short SHA(s) that drove this change}
```

Changelog rules:
- Entries are **append-only** — never modify or remove existing entries
- The requirements section must contain only current-state descriptions (no "was previously X")
- History lives exclusively in the changelog

### Plan Updates (context/plans/ or context/sites/ for legacy)
- If the plan missed a task, add it with proper T- prefix and dependencies
- If the plan had incorrect sequencing, fix the dependency graph
- Update plan-known-issues.md if the fix reveals a systemic issue

### Source-Tree CLAUDE.md Updates
- If the fix affects a module whose `CLAUDE.md` doesn't reference the relevant cavekit requirement, add the reference
- If a new source directory was created, create a `CLAUDE.md` with the appropriate cavekit references

### DESIGN.md Updates (project root)

If the fix involves visual changes and `DESIGN.md` exists:
- If the fix reveals a missing design pattern, add it to the appropriate section
- If the fix reveals an incorrect token value, update it
- Log all changes to `context/designs/design-changelog.md`:
  ```markdown
  | {date} | Section {N} | {what changed} | /ck:revise (commit {SHA}) |
  ```
- Update the `last_edited` frontmatter date in DESIGN.md

### Impl Tracking Updates (context/impl/)
- Record the manual fix in the relevant impl tracking file
- Add to the "Dead Ends & Failed Approaches" section if the fix replaced a failed approach
- Update test health if new tests were added

## Step 4b: Update Cavekit Overview (R3: Overview Consistency)

After updating individual kits, update `context/kits/cavekit-overview.md`:

1. **Requirement counts** — Re-count requirements in each updated cavekit and update the Domain Index table
2. **Domain descriptions** — If a cavekit's `## Scope` changed, update the overview's description column to match
3. **Cross-reference map** — If cross-references were added/removed/changed, update the map
4. **Frontmatter** — Update `last_edited` on the overview and all modified cavekit files

Only update the overview if individual kits were actually modified. If no kits changed, skip this step.

## Step 4c: Detect Build Site Drift (R4: Drift Detection)

After updating kits, scan the build site(s) in `context/plans/` (or `context/sites/` for legacy projects) and report any drift:

1. **Stale tasks** — Tasks whose parent requirement text no longer matches the cavekit (show old vs new)
2. **Orphaned tasks** — Tasks whose parent requirement was removed entirely
3. **Out-of-date tasks** — Tasks whose acceptance criteria changed in the cavekit

Output a drift report:

```markdown
### Build Site Drift
| Task | Build Site | Issue | Old | New |
|------|------------|-------|-----|-----|
| T-{n} | {site file} | stale/orphaned/out-of-date | {old text} | {new text} |
```

**Do NOT auto-modify the build site.** Only report drift. The user or architect decides whether to regenerate affected sections.

## Step 5: Run Tests

Run the project's test suite to verify that:
- The manual fixes still pass
- No regressions were introduced by context file updates
- Any new acceptance criteria added in Step 4 have corresponding tests

If the build command is not obvious, ask the user for `{BUILD_COMMAND}` and `{TEST_COMMAND}`.

## Step 6: Report

Generate a summary report:

```markdown
## Revision Report

### Commits Analyzed
| Commit | Category | Action |

### Manual Fixes Traced
| Commit | WHAT | WHY | RULE | LAYER | Files Updated |

### Context Files Updated
- kits: {list of updated cavekit files}
- DESIGN.md: {list of design system updates, or "no visual changes"}
- plans: {list of updated plan files}
- impl: {list of updated impl files}
- CLAUDE.md: {list of source-tree CLAUDE.md files created or updated}

### Test Results
- Pass: {count}
- Fail: {count}
- New tests added: {count}

### Recommendations
- {Any systemic prompt changes suggested}
- {Any kits that need deeper review}
```

Present this report to the user.
