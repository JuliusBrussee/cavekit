---
name: ck-init
description: "Bootstrap the context hierarchy — creates all directories, CLAUDE.md files, and index files"
---

> **Note:** `/bp:init` is deprecated and will be removed in a future version. Use `/ck:init` instead.

# Cavekit Init — Bootstrap Context Hierarchy

Creates the full context hierarchy for a Cavekit project. Run once at the start of a project, or re-run safely — it only creates what's missing.

## Properties

- **Idempotent** — creates only what's missing. Safe to re-run.
- **Non-destructive** — never overwrites existing files.
- **No questions** — does not ask what you're building (that's `/ck:sketch`).

## Step 1: Scan Existing Project Structure

Detect top-level directories that contain source code or other project artifacts:
- `src/`, `lib/`, `app/`, `pkg/`, `cmd/`, `internal/` (source directories)
- `tests/`, `test/`, `spec/` (test directories)
- `scripts/` (utility scripts)
- Any other directories that contain code files (`.ts`, `.js`, `.py`, `.go`, `.rs`, `.java`, etc.)

Record which directories exist — you'll create CLAUDE.md files for them in Step 3.

## Step 2: Create Context Directories

Create these directories if they don't exist:
- `context/`
- `context/refs/`
- `context/kits/`
- `context/designs/`
- `context/plans/`
- `context/impl/`
- `context/impl/archive/`

## Step 3: Create CLAUDE.md Files

Create each file below **only if it does not already exist**. Never overwrite.

### `context/CLAUDE.md`

```markdown
# Context Hierarchy

This project uses Cavekit's context hierarchy.

## Tiers
- refs/ — Source material (Tier 1: what IS). Read-only.
- kits/ — Requirements (Tier 2: what MUST BE). Start at cavekit-overview.md.
- designs/ — Visual design system (cross-cutting constraint). Start at DESIGN.md.
- plans/ — Task graphs (Tier 3: HOW). Start at plan-overview.md.
- impl/ — Progress tracking (Tier 4: what WAS DONE). Start at impl-overview.md.

## Navigation
Start at the overview file in whichever tier is relevant to your task.
Only load domain-specific files when the overview points you there.
For UI work, always read DESIGN.md at the project root first.
```

### `context/refs/CLAUDE.md`

```markdown
# Reference Materials

Source of truth that kits are derived from. Read-only.

## Conventions
- Organized by source in subdirectories (e.g., prd/, old-code-docs/, api-spec/)
- Agents read but never modify these files
- Kits reference specific files and sections via file:line
```

### `context/kits/CLAUDE.md`

```markdown
# Kits

Kits define WHAT needs implementing. They are implementation-agnostic.

## Conventions
- Start with cavekit-overview.md for the domain index
- R-numbered requirements (R1, R2, R3...)
- Every requirement has testable acceptance criteria
- Never prescribe HOW — that belongs in plans/
- Cross-reference related domains
- Decompose into subdirectories when a domain covers multiple independent concerns
```

### `context/plans/CLAUDE.md`

```markdown
# Plans

Plans define HOW to implement kits. They contain task dependency graphs.

## Conventions
- Start with plan-overview.md for the build site index
- Build sites use T-numbered tasks organized into dependency tiers
- Each task references cavekit requirements by ID
- build-site.md is the primary build site
- build-site-{feature}.md for feature-specific sites
```

### `context/impl/CLAUDE.md`

```markdown
# Implementation Tracking

Impls record what was built, what is pending, what failed.

## Conventions
- Start with impl-overview.md for current status across all domains
- impl-{domain}.md for per-domain tracking
- dead-ends.md for failed approaches (critical — prevents retrying failures)
- archive/ for compacted history
- Update after every implementation session
```

### `context/designs/CLAUDE.md`

```markdown
# Design System

The project's visual design system in DESIGN.md format (9-section Google Stitch).

## Conventions
- DESIGN.md at project root is the canonical source
- All UI implementation must reference DESIGN.md tokens and patterns
- Updated via /ck:design or automatically during /ck:check and /ck:revise
- Agents read this before implementing any user-facing component
- design-changelog.md tracks all design system changes
```

### Source-tree CLAUDE.md files

For each detected source directory from Step 1 (e.g., `src/`, `tests/`, `scripts/`), create a minimal CLAUDE.md **only if one does not already exist**:

**`src/CLAUDE.md`** (or equivalent source directory):
```markdown
# Source Code

See context/kits/ for requirements this code implements.
See context/plans/ for task dependency graphs.
```

**`tests/CLAUDE.md`** (or equivalent test directory):
```markdown
# Tests

Test conventions for this project.
See context/impl/ for test health tracking.
```

**`scripts/CLAUDE.md`** (if scripts/ exists):
```markdown
# Scripts

Utility scripts for this project.
```

## Step 4: Create Empty Index Files

Create these **only if they don't already exist**:

### `context/kits/cavekit-overview.md`

```markdown
---
created: "{CURRENT_DATE_UTC}"
last_edited: "{CURRENT_DATE_UTC}"
---

# Cavekit Overview

## Project
{Project name — read from README.md, package.json, or directory name}

## Domain Index
| Domain | File | Summary | Status |
|--------|------|---------|--------|

## Cross-Reference Map
| Domain A | Interacts With | Interaction Type |
|----------|---------------|-----------------|

## Dependency Graph
No domains defined yet. Run `/ck:sketch` to create kits.
```

### `context/plans/plan-overview.md`

```markdown
---
created: "{CURRENT_DATE_UTC}"
last_edited: "{CURRENT_DATE_UTC}"
---

# Plan Overview

## Build Sites
| Site | File | Tasks | Done | Status |
|------|------|-------|------|--------|

No build sites yet. Run `/ck:map` to generate one.
```

### `context/impl/impl-overview.md`

```markdown
---
created: "{CURRENT_DATE_UTC}"
last_edited: "{CURRENT_DATE_UTC}"
---

# Implementation Overview

## Domain Status
| Domain | Tasks Done | Tasks Total | Status |
|--------|-----------|-------------|--------|

No implementations tracked yet. Run `/ck:make` to start building.
```

## Step 5: Detect Legacy Layout

If `context/sites/` exists and contains build site files:

```
Found context/sites/ (legacy layout). Migrate to context/plans/?
This moves build-site files and updates internal references.
[Y/n]
```

If the user accepts:
1. Move all `*.md` files from `context/sites/` to `context/plans/` (skip `archive/` subdirectory)
2. If `context/sites/archive/` exists, move it to `context/impl/archive/` (merge, don't overwrite)
3. Remove `context/sites/` if now empty
4. Report which files were moved

If the user declines:
- Log: "Keeping context/sites/. Cavekit commands will check both locations."
- Do NOT migrate. The fallback is permanent.

## Step 6: Report and Commit

Report what was created:

```markdown
## Init Report

### Directories Created
- {list of new directories}

### CLAUDE.md Files Created
- {list of new CLAUDE.md files}

### Index Files Created
- {list of new index files}

### Legacy Migration
- {migration status if applicable}

### Next Step
Run `/ck:sketch` to start writing kits.
```

Then commit the scaffolding with message: "Initialize Cavekit context hierarchy"
