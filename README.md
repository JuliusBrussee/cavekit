<p align="center">
  <img src=".codex-plugin/logo.svg" alt="Cavekit" height="80">
</p>

<h1 align="center">cavekit</h1>

<p align="center">
  <strong>why agent guess when agent can know</strong>
</p>

<p align="center">
  <a href="https://github.com/JuliusBrussee/cavekit/stargazers"><img src="https://img.shields.io/github/stars/JuliusBrussee/cavekit?style=flat&color=yellow" alt="Stars"></a>
  <a href="https://github.com/JuliusBrussee/cavekit/commits/main"><img src="https://img.shields.io/github/last-commit/JuliusBrussee/cavekit?style=flat" alt="Last Commit"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/JuliusBrussee/cavekit?style=flat" alt="License"></a>
  <a href="https://docs.anthropic.com/en/docs/claude-code"><img src="https://img.shields.io/badge/Claude_Code-plugin-blueviolet" alt="Claude Code Plugin"></a>
</p>

<p align="center">
  <a href="#install">Install</a> •
  <a href="#before--after">Before/After</a> •
  <a href="#how-it-works">How It Works</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#parallel-execution">Parallel Execution</a> •
  <a href="#codex-adversarial-review">Codex Review</a> •
  <a href="#commands">Commands</a> •
  <a href="example.md">Examples</a>
</p>

<p align="center">
  Part of the <a href="https://github.com/JuliusBrussee/caveman">Caveman</a> ecosystem
</p>

---

A [Claude Code](https://docs.anthropic.com/en/docs/claude-code) plugin that turns natural language into **specs**, specs into **parallel build plans**, and build plans into **working software** — with automated iteration, validation, and dual-model adversarial review.

You describe what you want. Cavekit writes the contract. Agents build from the contract. Every line of code traces to a requirement. Every requirement has acceptance criteria. Nothing gets lost, nothing gets guessed.

## Before / After

<table>
<tr>
<td width="50%">

### Without Cavekit

```
> Build me a task management API

  (agent writes 2000 lines)
  (no tests)
  (forgot the auth middleware)
  (wrong database schema)
  (you spend 3 hours fixing it)
```

One shot. No validation. No traceability.
The agent guessed what you wanted.

</td>
<td width="50%">

### With Cavekit

```
> /bp:draft
  4 kits, 22 requirements, 69 criteria

> /bp:architect
  34 tasks across 5 dependency tiers

> /bp:build
  18 iterations — each validated against
  the spec before committing

  CAVEKIT COMPLETE
```

Every requirement traced. Every criterion checked.

</td>
</tr>
</table>

**Same feature. Zero guesswork. Full traceability.**

---

## The Problem

AI coding agents are powerful, but they fail the same way every time:

| Failure | What Happens |
|---------|-------------|
| **Context loss** | Agent forgets what it said three steps ago |
| **No validation** | Code written, never verified against intent |
| **No parallelism** | One agent, one task, one branch — even when work is independent |
| **No iteration** | Single pass produces a rough draft, not production code |

Cavekit fixes all four.

---

## The Idea

Instead of "prompt and pray," Cavekit puts a **specification layer** between your intent and the code.

```
                        ┌─── Task 1 ─── Agent A ───┐
                        │                           │
You ── /bp:draft ──► Kits ── /bp:architect ──► Build Site ──┤─── Task 2 ─── Agent B ───┤──► done
                        │                           │
                        └─── Task 3 ─── Agent C ───┘
```

Kits are the source of truth. Agents read them, build from them, validate against them. When something breaks, the system traces the failure back to the kit — not the code.

Spec is the product. Code is the derivative.

---

## Install

```bash
git clone https://github.com/JuliusBrussee/cavekit.git ~/.cavekit
cd ~/.cavekit && ./install.sh
```

Registers the plugin with Claude Code, syncs into Codex marketplace, installs the `cavekit` CLI. Restart Claude Code after installing.

**Requires:** [Claude Code](https://docs.anthropic.com/en/docs/claude-code), git, macOS/Linux.

**Optional:** [Codex](https://github.com/openai/codex) (`npm install -g @openai/codex`) — adds adversarial review. Cavekit works without it. Codex makes it significantly harder to ship flawed specs and broken code.

---

## How It Works

Four phases. Each one a slash command.

```
  RESEARCH         DRAFT            ARCHITECT           BUILD              INSPECT
  ────────         ─────            ─────────           ─────              ───────
  (optional)       "What are we     Break into tasks,   Auto-parallel:     Gap analysis:
  Multi-agent       building?"      map dependencies,    /bp:build          built vs.
  codebase +                        organize into        groups work        intended.
  web research     Produces:        tiered build site    into adaptive      Peer review.
                   kits with        + dependency graph   subagent packets   Trace to specs.
  Produces:        R-numbered                            tier by tier
  research brief   requirements     Produces:                               Produces:
                                    task graph           Codex reviews      findings report
                   Codex challenges                      every tier gate
                   the design
```

### 0. Research — ground the design (optional)

```
/bp:research "build a C+ compiler"
```

Dispatches 2–8 parallel subagents to explore the codebase and search the web for best practices, library landscape, reference implementations, and common pitfalls. A synthesizer agent cross-validates findings and produces a research brief in `context/refs/`.

### /bp:design — establish the design system

```
/bp:design
```

Creates or imports a **DESIGN.md** design system — a cross-cutting constraint layer enforced across the entire pipeline. Every kit references its design tokens, every task carries a Design Ref, every build result is audited for violations.

| Sub-command | What it does |
|------------|-------------|
| `/bp:design create` | Generate new DESIGN.md via guided Q&A |
| `/bp:design import` | Extract DESIGN.md from existing codebase |
| `/bp:design audit` | Check implementation against DESIGN.md |
| `/bp:design update` | Revise DESIGN.md, log to changelog |

### 1. Draft — define the what

```
/bp:draft
```

Describe what you're building in natural language. Cavekit decomposes it into **domain kits** — structured documents with numbered requirements (R1, R2, ...) and testable acceptance criteria. Stack-independent. Human-readable.

After internal review, kits go to Codex for a [design challenge](#design-challenge--catch-spec-flaws-before-building) — adversarial review that catches decomposition flaws, missing requirements, and ambiguous criteria before any code is written.

For existing codebases: `/bp:draft --from-code` reverse-engineers kits from your code and identifies gaps.

### 2. Architect — plan the order

```
/bp:architect
```

Reads all kits. Breaks requirements into tasks. Maps dependencies. Organizes into a **tiered build site** — a dependency graph where Tier 0 has no deps, Tier 1 depends only on Tier 0, and so on. Includes a **Coverage Matrix** mapping every acceptance criterion to its task(s). Nothing specified gets lost in translation.

### 3. Build — run the loop

```
/bp:build
```

Pre-flight coverage check validates all acceptance criteria are covered. Then the loop runs:

```
  ┌──────────────────────────────────────────────────────┐
  │                                                      │
  │  Read build site → Find next unblocked task          │
  │       │                                              │
  │       ▼                                              │
  │  Load relevant kit + acceptance criteria             │
  │       │                                              │
  │       ▼                                              │
  │  Implement the task                                  │
  │       │                                              │
  │       ▼                                              │
  │  Validate (build + tests + acceptance criteria)      │
  │       │                                              │
  │       ├── PASS → commit → mark done → next ──┐      │
  │       │                                       │      │
  │       └── FAIL → diagnose → fix → revalidate  │      │
  │                                               │      │
  │  ◄────────────────────────────────────────────┘      │
  │                                                      │
  │  Loop until: all tasks done OR limit reached         │
  └──────────────────────────────────────────────────────┘
```

At every tier boundary, [Codex adversarial review](#codex-adversarial-review) gates advancement. P0/P1 findings must be fixed before the next tier starts. With speculative review (default), this adds near-zero latency.

Post-flight verification cross-references what was built against original kits. Gaps get remediation tasks.

### 4. Inspect — verify the result

```
/bp:inspect
```

Gap analysis: built vs. specified. Peer review: bugs, security, missed requirements. Everything traced back to kit requirements.

---

## Quick Start

**Greenfield:**

```
> /bp:draft
What are you building?

> A REST API for task management. Users, projects, tasks
  with priorities and due dates. PostgreSQL.

Created 4 kits (22 requirements, 69 acceptance criteria)
Next: /bp:architect

> /bp:architect
Generated build site: 34 tasks, 5 tiers
Next: /bp:build

> /bp:build
Loop activated — 34 tasks, 20 max iterations.
...
All tasks done. Build passes. Tests pass.
CAVEKIT COMPLETE — 34 tasks in 18 iterations.
```

**Existing codebase:**

```
> /bp:draft --from-code
Exploring codebase... Next.js 14, Prisma, NextAuth.
Created 6 kits — 4 requirements are gaps (not yet implemented).

> /bp:architect --filter collaboration
Generated build site: 8 tasks, 3 tiers

> /bp:build
CAVEKIT COMPLETE — 8 tasks in 8 iterations.
```

See [example.md](example.md) for full annotated sessions.

---

## Parallel Execution

`/bp:build` parallelizes automatically. Multiple ready tasks get grouped into coherent work packets and dispatched concurrently.

```
═══ Wave 1 ═══
3 task(s) ready:
  T-001: Database schema       (tier 0, deps: none)
  T-002: Auth middleware        (tier 0, deps: none)
  T-003: Config loader          (tier 0, deps: none)

Dispatching 2 grouped subagents...
All 3 tasks complete. Merging...

═══ Wave 2 ═══
2 task(s) ready:
  T-004: User endpoints         (tier 1, deps: T-001, T-002)
  T-005: Health check           (tier 1, deps: T-003)

Dispatching 2 grouped subagents...
All done.

═══ BUILD COMPLETE ═══
Waves: 2 | Tasks: 5/5
```

| Step | What happens |
|------|-------------|
| **Compute frontier** | Find all tasks whose dependencies are complete |
| **Group** | Bundle frontier into work packets by shared files, subsystem, task size |
| **Dispatch** | Run packets as parallel subagents |
| **Merge** | Collect results, compute next frontier |
| **Repeat** | Wave-by-wave until all tasks done — no manual intervention |

Circuit breakers prevent infinite loops: 3 test failures → task BLOCKED, all blocked → stop and report.

---

## Codex Adversarial Review

Cavekit uses [Codex](https://github.com/openai/codex) as an adversarial reviewer — a second model with different training and different blind spots. Catches things Claude cannot see in its own output. Operates at three levels:

### Design Challenge — catch spec flaws before building

After kits are drafted and internally reviewed, the full set goes to Codex:

```
Claude drafts            Kit set             Codex challenges         User reviews
kits ──────► reviewer approves ──────► the design ──────► kits + findings
```

| Finding type | Behavior |
|-------------|----------|
| **Critical** | Must fix before building. Auto-fix loop, up to 2 cycles |
| **Advisory** | Presented alongside kits at user review gate |

No implementation feedback allowed. No framework suggestions. Only design-level concerns that would cause real problems during build.

### Tier Gate — catch code defects between tiers

Every completed tier triggers a Codex code review before advancing:

```
═══ Tier 0 Complete ═══
Codex reviews diff (T-001, T-002, T-003) ...
Review: 2 findings (1 P0, 1 P3)
Gate: BLOCKED → fix cycle 1/2

Fixing P0: nil pointer in auth middleware ...
Re-review ...
Gate: PROCEED

═══ Tier 1 starting ═══
```

| Severity | Behavior |
|----------|----------|
| **P0** (critical) | Blocks advancement. Auto-generates fix task |
| **P1** (high) | Blocks advancement. Auto-generates fix task |
| **P2** (medium) | Logged, does not block |
| **P3** (low) | Logged, does not block |

Gate modes: `severity` (default — P0/P1 block), `strict` (all block), `permissive` (nothing blocks), `off`.

Fix cycle runs up to 2 iterations per tier. After that, advances with warning. Never deadlocks.

### Speculative Review — eliminate gate latency

Codex reviews the *previous* tier in the background while Claude builds the *current* tier:

```
Tier 0 complete ───────────────────────────► Tier 1 complete
     │                                            │
     └── Codex reviews Tier 0 (background) ──────►│
                                                   │
                         Results ready ◄───────────┘
                         before gate runs
```

Results are already available when the gate checks. Near-zero latency. Falls back to synchronous if needed.

### Command Safety Gate

PreToolUse hook intercepts every Bash command before execution:

```
Agent runs command
     │
     ▼
Fast-path check ──► allowlist (50+ safe commands) → approve
     │           └► blocklist (rm -rf, force push, DROP TABLE) → block
     │
     ▼ (ambiguous)
Codex classifies ──► safe / warn / block
     │
     ▼ (cached)
Verdict cache ──► normalized pattern → reuse verdict
```

Integrates with Claude Code's permission system. Cached per session. Falls back to static rules when Codex is unavailable — never blocks solely because classifier is unreachable.

### Graceful Degradation

All Codex features are **additive**. Without Codex installed:

| Feature | Fallback |
|---------|----------|
| Design challenge | Skipped — internal reviewer still runs |
| Tier gate | Skipped — build proceeds without review pauses |
| Command gate | Static allowlist/blocklist only |

Cavekit works the same. Codex makes it harder to ship bad specs and bad code.

---

## Configuration

Settings live in two places:

| Location | Scope |
|----------|-------|
| `~/.cavekit/config` | User default |
| `.cavekit/config` | Project override (takes precedence) |

| Setting | Values | Default | Purpose |
|---------|--------|---------|---------|
| `bp_model_preset` | `expensive` `quality` `balanced` `fast` | `quality` | Model selection for Cavekit commands |
| `codex_review` | `auto` `off` | `auto` | Enable/disable Codex reviews |
| `codex_model` | model string | (Codex default) | Model for Codex calls |
| `tier_gate_mode` | `severity` `strict` `permissive` `off` | `severity` | How findings gate tier advancement |
| `command_gate` | `all` `interactive` `off` | `all` | Which sessions get command gating |
| `command_gate_timeout` | milliseconds | `3000` | Codex safety classification timeout |
| `speculative_review` | `on` `off` | `on` | Background review of previous tier |
| `speculative_review_timeout` | seconds | `300` | Max wait for speculative results |

**Model presets:**

| Preset | Reasoning | Execution | Exploration |
|--------|-----------|-----------|-------------|
| `expensive` | `opus` | `opus` | `opus` |
| `quality` | `opus` | `opus` | `sonnet` |
| `balanced` | `opus` | `sonnet` | `haiku` |
| `fast` | `sonnet` | `sonnet` | `haiku` |

```
/bp:config                      # show current
/bp:config preset balanced      # change preset
/bp:config preset fast --global # change default
```

---

## Commands

### Claude Code

| Command | Phase | What it does |
|---------|-------|-------------|
| `/bp:research` | Research | Multi-agent codebase + web research, produces brief |
| `/bp:design` | Design | Create, import, audit, or update DESIGN.md |
| `/bp:draft` | Draft | Decompose requirements into domain kits |
| `/bp:architect` | Architect | Generate tiered build site from kits |
| `/bp:build` | Build | Auto-parallel build with validation loop |
| `/bp:inspect` | Inspect | Gap analysis + peer review against kits |
| `/bp:config` | — | Show or update execution preset |
| `/bp:codex-review` | — | Standalone Codex adversarial review on diff |
| `/bp:progress` | — | Check build site progress |
| `/bp:gap-analysis` | — | Compare built vs. intended |
| `/bp:revise` | — | Trace manual fixes back into kits |
| `/bp:help` | — | Usage guide |

### CLI

| Command | What it does |
|---------|-------------|
| `cavekit monitor` | Interactive launcher — pick build sites, launch in tmux |
| `cavekit status` | Show build site progress |
| `cavekit kill` | Stop all sessions, clean up worktrees |
| `cavekit version` | Print version |
| `cavekit debug` | Show state file path and version |
| `cavekit reset` | Clear persisted state |

---

## File Structure

```
context/
├── kits/                     # Domain kits (persist across cycles)
│   ├── kit-overview.md
│   └── kit-{domain}.md
├── designs/                  # Design system artifacts
│   ├── DESIGN.md
│   └── design-changelog.md
├── sites/                    # Build sites (one per plan)
│   └── build-site-*.md
├── impl/                     # Implementation tracking
│   ├── impl-{domain}.md
│   ├── impl-review-findings.md
│   ├── impl-speculative-log.md
│   └── loop-log.md
└── refs/                     # Research briefs + raw findings
```

---

## Methodology

Cavekit applies the **scientific method** to AI-generated code. LLMs are non-deterministic. Software engineering doesn't have to be.

| Concept | Role |
|---------|------|
| **Kits** | The hypothesis — what you expect the software to do |
| **Validation gates** | Controlled conditions — build, tests, acceptance criteria |
| **Convergence loops** | Repeated trials — iterate until stable |
| **Implementation tracking** | Lab notebook — what was tried, what worked, what failed |
| **Revision** | Update the hypothesis — trace bugs back to kits |

Ships with 9 specialized agents (including **design-reviewer** for UI validation against DESIGN.md), a multi-agent research system, and 15 skills covering the full methodology. With Codex, operates as a **dual-model architecture** — Claude builds, Codex reviews — catching errors single-model self-review cannot.

<details>
<summary><strong>All 15 skills</strong></summary>

| Skill | What it covers |
|-------|---------------|
| [Design System](skills/design-system) | Create and maintain DESIGN.md |
| [UI Craft](skills/ui-craft) | Component patterns, animation, accessibility, review checklist |
| [Cavekit Writing](skills/cavekit-writing) | Write kits agents can consume |
| [Convergence Monitoring](skills/convergence-monitoring) | Detect when iterations plateau |
| [Peer Review](skills/peer-review) | Six modes for cross-model review |
| [Validation-First Design](skills/validation-first) | Every requirement must be verifiable |
| [Context Architecture](skills/context-architecture) | Progressive disclosure for agent context |
| [Revision](skills/revision) | Trace bugs upstream to kits |
| [Brownfield Adoption](skills/brownfield-adoption) | Add Cavekit to existing codebases |
| [Speculative Pipeline](skills/speculative-pipeline) | Overlap phases for faster builds |
| [Prompt Pipeline](skills/prompt-pipeline) | Design the prompts driving each phase |
| [Implementation Tracking](skills/impl-tracking) | Living records of build progress |
| [Documentation Inversion](skills/documentation-inversion) | Docs for agents, not just humans |
| [Peer Review Loop](skills/peer-review-loop) | Combine build loop with cross-model review |
| [Core Methodology](skills/methodology) | The full DABI lifecycle |

</details>

---

## Why "Cavekit"

Most AI coding tools treat the agent as a black box. Prompt, generate, hope. Cavekit inverts this.

**The spec is the product. The code is a derivative.**

When the spec is clear, the code follows. When the code is wrong, the spec tells you why. Without a specification, there's nothing to validate against. Cavekit gives every agent — current and future — a contract to build from and a standard to meet.

Two models disagreeing is a signal. Two models agreeing is confidence.

---

## Star This Repo

If cavekit save you mass debug time — leave star.

[![Star History Chart](https://api.star-history.com/svg?repos=JuliusBrussee/cavekit&type=Date)](https://star-history.com/#JuliusBrussee/cavekit&Date)

---

## Also by Julius Brussee

- **[Caveman](https://github.com/JuliusBrussee/caveman)** — Claude Code skill that cuts ~75% of output tokens. Same accuracy, way less fluff. `npx skills add JuliusBrussee/caveman`
- **[Revu](https://github.com/JuliusBrussee/revu-swift)** — local-first macOS study app with FSRS spaced repetition, decks, exams, and study guides. [revu.cards](https://revu.cards)

## License

MIT
