---
name: ck-help
description: Show Cavekit commands and usage
---

# Cavekit

## The Loop

```
/ck:sketch   →   /ck:map   →   /ck:make   →   /ck:check
 what to         how to        build it       verify
  build          build it                       it
```

Four commands. In that order. That is the whole main cycle.

```
one-shot:    /ck:ship "<describe your feature>"     # runs all four, no gates
```

## Core Commands

| Command | Phase | What it does |
|---------|-------|-------------|
| `/ck:sketch` | Draft | Decompose your project into domains; write kits with testable acceptance criteria |
| `/ck:map` | Architect | Read kits, generate a tiered build site with a task dependency graph |
| `/ck:make` | Build | Inline build loop — parent session implements tasks one at a time (no subagent, no worktree). Safe default. |
| `/ck:make-parallel` | Build (opt-in) | Parallel subagent build loop — fans out `ck:task-builder` subagents in isolated git worktrees. Throughput at the cost of harness-race surface. |
| `/ck:check` | Inspect | Gap analysis against kits + peer review of the code; produces APPROVE/REVISE/REJECT verdict |
| `/ck:ship` | End-to-end | One-shot sketch → map → make → check with no user gates. For tiny features and throwaways. |

## Auxiliary Commands

| Command | What it does | When to use it |
|---------|-------------|----------------|
| `/ck:init` | Bootstrap `context/` and `.cavekit/` state dir; detect capabilities | Once at project start, or re-run after installing a new tool |
| `/ck:design` | Create / import / update / audit DESIGN.md (visual system) | Before `/ck:sketch` when the project has UI |
| `/ck:research` | Parallel multi-agent research → research brief in `context/refs/` | Before `/ck:sketch` on novel domains |
| `/ck:revise` | Trace recent manual fixes back into kits; `--trace` runs the single-failure backprop protocol | After a hotfix or a review finding |
| `/ck:review` | Branch review — kit compliance + code quality; `--mode gap`, `--codex`, `--tier`, `--strict` narrow the scope | Before merging; as a tier gate inside `/ck:make`; on-demand second opinion |
| `/ck:status` | Progress report against the build site; `--watch` tails the live dashboard | During or after `/ck:make` |
| `/ck:team` | Initialize team mode, join a checkout, inspect claims/activity, and manage task ownership | Before multi-user Cavekit work in one repo |
| `/ck:config` | Show or update execution preset and runtime keys | Any time |
| `/ck:resume` | Recover an interrupted loop without losing progress | After a crash, lock conflict, or manual interrupt |
| `/ck:help` | This page | — |

## Flags at a Glance

```bash
/ck:sketch                        # interactive — asks what to build
/ck:sketch context/refs/          # from reference materials (PRDs, docs)
/ck:sketch --from-code            # reverse-engineer from an existing codebase
/ck:sketch --filter v2            # only v2 kits

/ck:map                           # generate build site from all kits
/ck:map --filter v2               # scope to v2

/ck:make                          # inline sequential build (default, no subagent)
/ck:make --filter v2              # scope
/ck:make --peer-review            # Codex tier gates
/ck:make --max-iterations 30      # iteration cap

/ck:make-parallel                 # opt-in: parallel ck:task-builder subagents in worktrees
/ck:make-parallel --concurrency 4 # set max concurrent packets per wave (default 3)

/ck:check                         # full inspection
/ck:check --filter v2             # scope

/ck:ship "<feature description>"  # one-shot
/ck:ship "add JWT auth" --skip-check
/ck:ship "refactor API" --peer-review

/ck:review                        # two-pass branch review
/ck:review --mode gap             # gap analysis only (replaces /ck:scan)
/ck:review --codex                # Codex-only adversarial review (replaces /ck:judge)
/ck:review --tier                 # tier-gate mode (emits fix tasks back into loop)
/ck:review --strict               # IMPORTANT also blocks
/ck:review --base main            # override diff base

/ck:revise                        # multi-commit revision sweep
/ck:revise --trace                # single-failure backprop (replaces /ck:backprop)
/ck:revise --trace --from-flag    # auto-consume .cavekit/.auto-backprop-pending.json
/ck:revise --trace --from-finding F-012

/ck:status                        # one snapshot
/ck:status --watch                # tail live dashboard (replaces /ck:watch)
/ck:status --watch --interval 5
/ck:status --team                 # active claims + recent team activity

/ck:team init                     # create .cavekit/team scaffolding
/ck:team join                     # persist local identity for this checkout
/ck:team claim T-013              # manually claim a task
/ck:team release T-013 --complete # mark claimed task complete and release it

/ck:init                          # full bootstrap
/ck:init --tools-only             # re-detect CLI tools, MCP servers, plugins (replaces /ck:setup-tools)
/ck:init --tools-only --summary-only

/ck:config                        # show current
/ck:config list                   # show built-in presets
/ck:config preset balanced        # set repo override
/ck:config preset fast --global   # set user default
```

## Model Presets

| Preset | Reasoning | Execution | Exploration |
|--------|-----------|-----------|-------------|
| `expensive` | opus | opus | opus |
| `quality` | opus | opus | sonnet |
| `balanced` | opus | sonnet | haiku |
| `fast` | sonnet | sonnet | haiku |

Precedence: `.cavekit/config` overrides `~/.cavekit/config`, which overrides the built-in default (`quality`).

## Skills (reference docs)

| Skill | Topic |
|-------|-------|
| `ck:methodology` | Core Hunt lifecycle |
| `ck:design-system` | How to write and maintain DESIGN.md |
| `ck:cavekit-writing` | How to write kits with testable criteria |
| `ck:peer-review` | Cross-model review patterns + Codex loop |
| `ck:validation-first` | Every requirement must be auto-testable |
| `ck:convergence-monitoring` | Detecting if loop is converging or stuck |
| `ck:revision` | Tracing bugs back to kits (includes automated backprop) |
| `ck:context-architecture` | Organizing `context/` for AI agents |
| `ck:impl-tracking` | Progress tracking and dead ends |
| `ck:brownfield-adoption` | Adopting Cavekit on existing codebases |
| `ck:prompt-pipeline` | Designing prompt sequences |
| `ck:speculative-pipeline` | Staggered pipeline execution |
| `ck:documentation-inversion` | Agent-first documentation |
| `ck:karpathy-guardrails` | Think-before-code, simplicity, surgical changes |
| `ck:autonomous-loop` | State machine, sentinels, lock protocol |
