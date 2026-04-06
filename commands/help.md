---
name: ck-help
description: Show Cavekit commands and usage
---

> **Note:** `/bp:help` is deprecated and will be removed in a future version. Use `/ck:help` instead.

# Cavekit

## The Workflow

```
/ck:init        →  bootstrap context hierarchy (optional, /ck:sketch does this too)
/ck:design      →  create or update DESIGN.md (the LOOK) — import, extract, or design interactively
/ck:research    →  deep multi-agent research (standalone, or integrated into /ck:sketch)
/ck:sketch       →  write kits (the WHAT) — offers research if warranted, references DESIGN.md
/ck:map   →  generate site (the ORDER) — includes design refs for UI tasks
/ck:make       →  ralph loop (the BUILD) — task builders read DESIGN.md for UI work
/ck:check     →  gap analysis + peer review + design compliance (the CHECK)
/ck:config      →  show or change execution model presets
```

### Quick Mode

```bash
/ck:quick <describe your feature>   # runs all 4 phases end-to-end, no stops
/ck:quick add JWT auth --skip-inspect
/ck:quick refactor the API layer --peer-review
```

Streamlined draft + architect (no interactive Q&A, no user gates) followed by the full build and inspect. Best for small-to-medium features where you trust the decomposition.

### Model Presets

```bash
/ck:config                         # show effective preset + resolved models
/ck:config list                    # show built-in presets
/ck:config preset balanced         # set repo override
/ck:config preset fast --global    # set your default for all repos
```

Built-in presets:

| Preset | Reasoning | Execution | Exploration |
|--------|-----------|-----------|-------------|
| `expensive` | `opus` | `opus` | `opus` |
| `quality` | `opus` | `opus` | `sonnet` |
| `balanced` | `opus` | `sonnet` | `haiku` |
| `fast` | `sonnet` | `sonnet` | `haiku` |

Precedence: `.cavekit/config` overrides `~/.cavekit/config`, which overrides the built-in default (`quality`).

## Commands

### `/ck:init` — Bootstrap Context Hierarchy

```bash
/ck:init                         # create all context dirs, CLAUDE.md files, index files
```

Creates the full context hierarchy for a Cavekit project. Idempotent — only creates what's missing. Detects legacy `context/sites/` layout and offers migration to `context/plans/`.

### `/ck:research` — Deep Research

```bash
/ck:research "build a Verse compiler targeting WASM"           # standard depth
/ck:research "add real-time collab" --depth deep               # exhaustive
/ck:research "refactor auth layer" --depth quick               # fast scan
/ck:research "new React dashboard" --web-only                  # greenfield, skip codebase
/ck:research "optimize DB queries" --codebase-only             # air-gapped, skip web
```

Runs parallel multi-agent research (codebase exploration + web search) and produces a research brief in `context/refs/research-brief-{topic}.md`. Dispatches 2-8 agents depending on project size and depth. Two-pass synthesis cross-validates findings and resolves contradictions.

Also integrated into `/ck:sketch` — when the draft phase detects a project that would benefit from research, it offers to run the pipeline inline before design Q&A.

### `/ck:design` — Create or Update DESIGN.md

```bash
/ck:design                        # interactive — collaborative design system creation
/ck:design --import claude        # import from awesome-design-md collection
/ck:design --import vercel        # import Vercel's design system as starting point
/ck:design --from-site https://...  # extract design system from a live site
/ck:design --audit                # quality check existing DESIGN.md
/ck:design --section 2            # update just Section 2 (Color Palette)
```

Creates the project's visual design system document following the 9-section Google Stitch format. DESIGN.md becomes the authoritative visual reference that all agents consult when building UI.

### `/ck:sketch` — Write Kits

```bash
/ck:sketch                        # interactive — asks what to build
/ck:sketch context/refs/          # from reference materials (PRDs, docs)
/ck:sketch --from-code            # from existing codebase (brownfield)
/ck:sketch --filter v2            # only generate v2 kits
```

Decomposes your project into domains, writes `context/kits/cavekit-{domain}.md` files with R-numbered requirements and testable acceptance criteria.

### `/ck:map` — Generate Site

```bash
/ck:map                    # generates site from all kits
/ck:map --filter v2        # only v2 kits
```

Reads kits, decomposes requirements into tasks, organizes into dependency tiers. Writes `context/plans/build-site.md`. No domain plans — just tasks and dependencies.

### `/ck:make` — Run the Loop

```bash
/ck:make                       # auto-parallel build from site
/ck:make --filter v2           # scope to v2
/ck:make --peer-review         # add Codex (GPT-5.4) review
/ck:make --max-iterations 30   # iteration limit
/ck:make --peer-review --codex-model gpt-5.4-mini
```

Auto-archives any previous cycle, then builds the site. Automatically parallelizes by grouping ready tasks into a few coherent work packets based on shared files, subsystem, and complexity. Progresses through tiers autonomously without manual intervention. If multiple build sites exist, asks which one to implement.

With `--peer-review`: alternates build and review iterations, calling Codex via MCP.

### `/ck:check` — Post-Loop Inspection

```bash
/ck:check                     # inspect everything from last loop
/ck:check --filter v2         # only v2
```

Runs after build completes. Does two things:
1. **Gap analysis** — compares built code against every cavekit requirement and acceptance criterion
2. **Peer review** — finds bugs, security issues, performance problems, quality gaps

Produces a verdict: APPROVE / REVISE / REJECT with prioritized findings.

### `/ck:config` — Execution Presets

```bash
/ck:config
/ck:config list
/ck:config preset quality
/ck:config preset fast --global
```

Shows or updates the active Cavekit execution preset. Presets map three task buckets:
`reasoning` for draft/architect/inspect-style work, `execution` for build/task-builder work, and `exploration` for research and codebase scanning helpers.

### `/ck:progress` — Check Progress

```bash
/ck:progress                    # show site progress
/ck:progress --filter v2
```

Shows tasks done/ready/blocked, progress bar, current tier, and next tasks.

### `/ck:judge` — On-Demand Codex Review

```bash
/ck:judge                  # review current tier diff
/ck:judge --base v1.0     # review diff against a specific ref
```

Sends the current diff to Codex for adversarial review. Outputs findings in Cavekit format and appends them to `context/impl/impl-review-findings.md`. Requires Codex CLI to be installed.

### Maintenance (optional)

| Command | When |
|---------|------|
| `/ck:judge` | On-demand Codex adversarial review of current diff |
| `/ck:scan` | After a loop — compare built vs intended |
| `/ck:revise` | After manual code fixes — trace back to kits |
| `/ck:compact-specs` | When impl tracking files exceed ~500 lines |
| `/ck:archive-loop` | Manually archive a loop cycle (build does this automatically) |
| `/ck:next-session` | Generate a handoff document for next session |

### Legacy (advanced)

These still work but are superseded by the three main commands:

| Command | Replaced by |
|---------|-------------|
| `/cavekit init` | `/ck:init` (or `/ck:sketch` creates directories automatically) |
| `/cavekit spec-from-refs` | `/ck:sketch context/refs/` |
| `/cavekit spec-from-code` | `/ck:sketch --from-code` |
| `/cavekit plan-from-specs` | `/ck:map` (generates site directly, no domain plans) |
| `/cavekit implement` | `/ck:make` (one task at a time vs full loop) |
| `/cavekit spec-loop` | `/ck:make` |
| `/cavekit peer-review-loop` | `/ck:make --peer-review` |
| `/cavekit quick` | `/ck:quick` (end-to-end with no stops) |

## Skills (reference docs)

| Skill | Topic |
|-------|-------|
| `ck:methodology` | Core DABI lifecycle |
| `ck:design-system` | How to write and maintain DESIGN.md (9-section Stitch format) |
| `ck:cavekit-writing` | How to write kits with testable criteria |
| `ck:peer-review` | Cross-model review patterns |
| `ck:peer-review-loop` | Ralph Loop + Codex architecture |
| `ck:validation-first` | Every requirement must be auto-testable |
| `ck:convergence-monitoring` | Detecting if loop is converging or stuck |
| `ck:revision` | Tracing bugs back to kits |
| `ck:context-architecture` | Organizing context/ for AI agents |
| `ck:impl-tracking` | Progress tracking and dead ends |
| `cavekit:brownfield-adoption` | Adopting Cavekit on existing codebases |
| `ck:prompt-pipeline` | Designing prompt sequences |
| `cavekit:speculative-pipeline` | Staggered pipeline execution |
| `cavekit:documentation-inversion` | Agent-first documentation |
