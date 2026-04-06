---
created: "2026-03-20T00:00:00Z"
last_edited: "2026-03-20T00:00:00Z"
---

# Cavekit: Spec Sync

## Scope
Making kits the living source of truth for the repository. Kits always reflect the current state of the system. Changes flow bidirectionally — specs drive builds, and code changes flow back into specs via `/ck:revise`. Each cavekit maintains a changelog tracking its evolution over time.

## Requirements

### R1: Spec-Aware Revision
**Description:** `/ck:revise` scans recent git commits (since last revision) and identifies which cavekit requirements are affected by the changes. It updates those requirements to reflect the current implementation, replacing outdated descriptions and acceptance criteria with what actually exists now.
**Acceptance Criteria:**
- [ ] Given commits that changed a module's behavior, running revise updates the relevant cavekit requirements to describe the new behavior
- [ ] Requirements not affected by recent changes remain untouched
- [ ] If a requirement is moved from one cavekit to another, cross-references in both kits are updated accordingly
- [ ] Revise determines "since last revision" from the most recent changelog entry's commit SHAs, falling back to all commits if no changelog exists
**Dependencies:** R2 (changelog entries record what was last revised)

### R2: Changelog Tracking
**Description:** Each cavekit maintains a `## Changelog` section at the bottom. Every revision appends an entry with the date, what changed, and which commits drove the change. The requirements section always reflects current truth — history lives only in the changelog.
**Acceptance Criteria:**
- [ ] Changelog entries include: ISO date, affected requirement IDs, summary of change, and commit SHAs
- [ ] Changelog is append-only — entries are never modified or removed
- [ ] The requirements section contains only current-state descriptions (no versioning, no "was previously X")
- [ ] New kits created via `/ck:sketch` start with an empty `## Changelog` section
**Dependencies:** none

### R3: Cavekit Overview Consistency
**Description:** When individual kits are revised, `cavekit-overview.md` is updated to reflect any changes to requirement counts, domain descriptions, or cross-reference maps.
**Acceptance Criteria:**
- [ ] After revise, the overview's requirement counts match the actual cavekit files
- [ ] Cross-reference map reflects current domain interactions
- [ ] Domain descriptions in the overview match the `## Scope` sections of individual kits
- [ ] The `last_edited` frontmatter date is updated on all modified files
**Dependencies:** R1

### R4: Build Site Drift Detection
**Description:** After revision, the system identifies tasks in the build site that no longer align with updated specs — flagging them as stale so the architect can regenerate affected portions.
**Acceptance Criteria:**
- [ ] Stale tasks are listed in the revise output with the specific drift (old requirement text vs new)
- [ ] Tasks whose parent requirement was removed are flagged as orphaned
- [ ] Tasks whose acceptance criteria changed are flagged as out-of-date
- [ ] Does not auto-modify the build site — only reports drift
**Dependencies:** R1

## Out of Scope
- Auto-triggering revise via hooks (it is always an explicit command)
- Modifying the build site automatically (that is the architect's responsibility)
- Tracking non-code changes (design files, external documentation)
- Brownfield reverse-engineering (that is `/ck:sketch --from-code`, a separate flow)

## Cross-References
- See also: cavekit-build-lifecycle.md (worktree changes must be merged to main before revise can scan them)
- See also: cavekit-site.md (drift detection reads the build site's task-to-requirement mappings)

## Changelog
