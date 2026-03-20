---
name: bp-revise
description: Trace recent manual code fixes back into blueprints and context files
---

# Blueprint Revise — Trace Fixes to Blueprints

You are performing revision: tracing recent manual code changes back into blueprints, plans, and context files so that the convergence loop can reproduce them autonomously. The core principle: when a fix exists only in code without a corresponding blueprint update, the iteration loop may reintroduce the same defect.

## Step 1: Analyze Recent Commits

Determine the revision range:
1. Read each blueprint's `## Changelog` section for the most recent entry's commit SHAs
2. Use the newest commit SHA across all blueprints as the "since" marker
3. If no changelog entries exist in any blueprint, fall back to `git log --oneline -20`
4. Run `git log --oneline <since>..HEAD` to gather commits since last revision

Read the diffs for each commit.

Classify each commit into one of three categories:

| Category | Description | Action |
|----------|-------------|--------|
| **Manual fix** | Human-authored code change fixing a bug or adding behavior | Revise (proceed to Step 2) |
| **Iteration loop** | Changes made by an automated convergence loop session | Skip — already blueprint-driven |
| **Infrastructure** | Build config, CI, tooling, dependency updates | Skip — not blueprint-relevant |

Report the classification table to the user.

## Step 2: Analyze Each Manual Fix

For each commit classified as a manual fix, determine:

1. **WHAT** changed — which files, which functions, what behavior was added/modified/removed
2. **WHY** it changed — from the commit message, PR description, diff context, and surrounding code comments
3. **RULE** — what invariant or requirement was violated that necessitated this fix
4. **LAYER** — which blueprint, plan, or prompt should have caught this:
   - **Blueprint gap**: the requirement was never specified
   - **Plan gap**: the blueprint existed but the plan didn't implement it
   - **Prompt gap**: the plan existed but the prompt didn't guide the agent to it
   - **Validation gap**: everything existed but no test caught the regression

## Step 3: Map Changes to Blueprint Requirements

For each manual fix, identify which blueprint requirements are affected:

1. **Read all blueprints** in `context/blueprints/` to build a requirement index (blueprint file → R-numbers → descriptions)
2. **Read the build site** in `context/sites/` to map tasks → requirements → blueprints
3. **Match changed files** to tasks in the build site (check task titles, impl tracking files, and git blame for task-ID references in commit messages)
4. **Identify affected requirements** by tracing: changed file → task → requirement → blueprint

For each affected requirement, record:
| Blueprint | Requirement | Current Description | What Changed | Needs Update? |

**Unaffected requirements remain untouched.** Only modify requirements where the code change reveals a gap, ambiguity, or behavioral shift.

**Cross-blueprint moves:** If a fix reveals that a requirement belongs in a different domain:
1. Move the requirement to the correct blueprint (assign the next available R-number)
2. Add a cross-reference in the original blueprint: "R{n} moved to blueprint-{domain}.md R{m}"
3. Update both blueprints' `## Cross-References` sections

## Step 3b: Discover Governing Plan Files

For each changed source file, determine which plan file governs it:
- Search `context/plans/` for plan files that reference the changed paths
- If no plan covers the file, flag it as an **untracked file** (potential blueprint gap)

## Step 4: Update Blueprints and Context Files

For each manual fix, update the appropriate context files:

### Blueprint Updates (context/blueprints/)
- If the fix reveals a missing requirement, add it with testable acceptance criteria
- If the fix reveals an ambiguous requirement, clarify it
- Update the `last_edited` frontmatter date to today's date
- **Append a changelog entry** to the blueprint's `## Changelog` section:

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

### Plan Updates (context/plans/)
- If the plan missed a task, add it with proper T- prefix and dependencies
- If the plan had incorrect sequencing, fix the dependency graph
- Update plan-known-issues.md if the fix reveals a systemic issue

### Impl Tracking Updates (context/impl/)
- Record the manual fix in the relevant impl tracking file
- Add to the "Dead Ends & Failed Approaches" section if the fix replaced a failed approach
- Update test health if new tests were added

## Step 4b: Update Blueprint Overview (R3: Overview Consistency)

After updating individual blueprints, update `context/blueprints/blueprint-overview.md`:

1. **Requirement counts** — Re-count requirements in each updated blueprint and update the Domain Index table
2. **Domain descriptions** — If a blueprint's `## Scope` changed, update the overview's description column to match
3. **Cross-reference map** — If cross-references were added/removed/changed, update the map
4. **Frontmatter** — Update `last_edited` on the overview and all modified blueprint files

Only update the overview if individual blueprints were actually modified. If no blueprints changed, skip this step.

## Step 4c: Detect Build Site Drift (R4: Drift Detection)

After updating blueprints, scan the build site(s) in `context/sites/` and report any drift:

1. **Stale tasks** — Tasks whose parent requirement text no longer matches the blueprint (show old vs new)
2. **Orphaned tasks** — Tasks whose parent requirement was removed entirely
3. **Out-of-date tasks** — Tasks whose acceptance criteria changed in the blueprint

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
- blueprints: {list of updated blueprint files}
- plans: {list of updated plan files}
- impl: {list of updated impl files}

### Test Results
- Pass: {count}
- Fail: {count}
- New tests added: {count}

### Recommendations
- {Any systemic prompt changes suggested}
- {Any blueprints that need deeper review}
```

Present this report to the user.
