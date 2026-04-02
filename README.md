<p align="center">
  <img src=".codex-plugin/logo.svg" alt="Blueprint" height="80">
</p>

<h3 align="center">Specification-driven development for AI coding agents</h3>

<p align="center">
  A Claude Code plugin that turns natural language into blueprints,<br>
  blueprints into parallel build plans, and build plans into working software —<br>
  with automated iteration, validation, and dual-model adversarial review via Codex.
</p>

<p align="center">
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
  <a href="https://docs.anthropic.com/en/docs/claude-code"><img src="https://img.shields.io/badge/Claude_Code-plugin-blueviolet" alt="Claude Code Plugin"></a>
  <img src="https://img.shields.io/badge/version-2.1.0-green" alt="Version 2.1.0">
</p>

<p align="center">
  <a href="#install">Install</a> &middot;
  <a href="#how-it-works">How It Works</a> &middot;
  <a href="#quick-start">Quick Start</a> &middot;
  <a href="#parallel-execution">Parallel Execution</a> &middot;
  <a href="#codex-adversarial-review">Codex Review</a> &middot;
  <a href="#commands">Commands</a> &middot;
  <a href="#methodology">Methodology</a> &middot;
  <a href="example.md">Examples</a>
</p>

---

## The Problem

AI coding agents are powerful, but they fail in predictable ways:

- **They lose context.** Ask an agent to build a full-stack feature and it forgets what it said three steps ago.
- **They skip validation.** Code gets written but never verified against the original intent.
- **They can't parallelize.** One agent, one task, one branch — even when the work is independent.
- **They don't iterate.** A single pass produces a rough draft, not production code.

Blueprint fixes all of this.

---

## The Idea

Instead of prompting an agent and hoping for the best, Blueprint introduces a **specification layer** between your intent and the code. You describe what you want. The system decomposes it into domain blueprints with numbered requirements and testable acceptance criteria. Then it builds from those blueprints — not from memory, not from vibes — in an automated loop that validates every step.

```
                        ┌─── Task 1 ─── Agent A ───┐
                        │                           │
You ── /bp:draft ──► Blueprints ── /bp:architect ──► Build Site ──┤─── Task 2 ─── Agent B ───┤──► done
                        │                           │
                        └─── Task 3 ─── Agent C ───┘
```

The blueprints are the source of truth. Agents read them, build from them, and validate against them. When something breaks, the system traces the failure back to the blueprint — not the code.

---

## Without Blueprint vs. With Blueprint

<table>
<tr><th width="50%">Without Blueprint</th><th width="50%">With Blueprint</th></tr>
<tr>
<td>

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
<td>

```
> /bp:draft
  4 blueprints, 22 requirements, 69 criteria

> /bp:architect
  34 tasks across 5 dependency tiers

> /bp:build
  18 iterations — each validated against
  the blueprint before committing

  BLUEPRINT COMPLETE
```

Every line of code traces to a requirement.
Every requirement has acceptance criteria.

</td>
</tr>
</table>

---

## Install

```bash
git clone https://github.com/JuliusBrussee/blueprint.git ~/.blueprint
cd ~/.blueprint && ./install.sh
```

This registers the Blueprint plugin with Claude Code, syncs it into your local Codex plugin marketplace, links Codex prompt files into `~/.codex/prompts/`, and installs the `blueprint` CLI. Restart Claude Code and Codex after installing.

**Requirements:** [Claude Code](https://docs.anthropic.com/en/docs/claude-code), git, macOS/Linux.

**Optional:** [Codex](https://github.com/openai/codex) (`npm install -g @openai/codex`) — enables adversarial review at the design, build, and command levels. Blueprint works without it, but Codex makes it significantly harder to ship flawed specs and broken code.

---

## How It Works

Blueprint follows four phases — **Draft, Architect, Build, Inspect** — each driven by a slash command inside Claude Code. An optional **Research** phase grounds the design in real evidence before blueprints are written.

```
  RESEARCH         DRAFT            ARCHITECT           BUILD                INSPECT
  ────────         ─────            ─────────           ─────                ───────
  (optional)       "What are we     Break into tasks,   Auto-parallel:       Gap analysis:
  Multi-agent       building?"      map dependencies,    /bp:build            built vs.
  codebase +                        organize into        groups work          intended.
  web research     Produces:        tiered build site    into adaptive        Peer review.
                   blueprints       + dependency graph   subagent packets     Trace to specs.
  Produces:        with R-numbered                       tier by tier
  research brief   requirements     Produces:                                 Produces:
  in context/refs                   task graph           Codex reviews        findings report
                   Codex challenges                      every tier gate
                   the design                            (speculative +
                                                         synchronous)
```

### 0. Research — ground the design (optional)

```
/bp:research "build a Verse compiler targeting WASM"
```

Dispatches 2–8 parallel subagents to explore the codebase and search the web for current best practices, library landscape, reference implementations, and common pitfalls. A synthesizer agent cross-validates findings and produces a research brief in `context/refs/`. Research is also offered inline during `/bp:draft` when the project involves unfamiliar technology or architectural decisions with multiple viable approaches.

### 1. Draft — define the what

```
/bp:draft
```

You describe what you're building in natural language. Blueprint decomposes it into **domain blueprints** — structured documents with numbered requirements (R1, R2, ...) and testable acceptance criteria. Each blueprint is stack-independent and human-readable.

When the project would benefit from it, the draft phase offers to run [deep research](#0-research--ground-the-design-optional) before design Q&A — grounding clarifying questions and approach proposals in real evidence rather than LLM priors.

After the internal reviewer approves, blueprints are sent to Codex for a [design challenge](#design-challenge--catch-spec-flaws-before-building) — an adversarial review that catches decomposition flaws, missing requirements, and ambiguous criteria before any code is written.

For existing codebases, `/bp:draft --from-code` reverse-engineers blueprints from your code and identifies gaps.

### 2. Architect — plan the order

```
/bp:architect
```

Reads all blueprints, breaks requirements into tasks, maps dependencies, and organizes everything into a **tiered build site** — a dependency graph where Tier 0 has no dependencies, Tier 1 depends only on Tier 0, and so on. This is what the build loop consumes.

### 3. Build — run the loop

```
/bp:build
```

The Ralph Loop. Each iteration:

```
  ┌──────────────────────────────────────────────────────────┐
  │                                                          │
  │  Read build site → Find next unblocked task              │
  │       │                                                  │
  │       ▼                                                  │
  │  Load relevant blueprint + acceptance criteria           │
  │       │                                                  │
  │       ▼                                                  │
  │  Implement the task                                      │
  │       │                                                  │
  │       ▼                                                  │
  │  Validate (build + tests + acceptance criteria)          │
  │       │                                                  │
  │       ├── PASS → commit → mark done → next task ──┐     │
  │       │                                            │     │
  │       └── FAIL → diagnose → fix → revalidate      │     │
  │                                                    │     │
  │  ◄─────────────────────────────────────────────────┘     │
  │                                                          │
  │  Loop until: all tasks done OR iteration limit reached   │
  └──────────────────────────────────────────────────────────┘
```

At every tier boundary, [Codex adversarial review](#codex-adversarial-review) gates advancement — P0/P1 findings must be fixed before the next tier starts. With speculative review enabled (default), this adds near-zero latency because the review runs in the background while the next tier builds.

### 4. Inspect — verify the result

```
/bp:inspect
```

Gap analysis compares what was built against what was specified. Peer review checks for bugs, security issues, and missed requirements. Everything traced back to blueprint requirements.

---

## Quick Start

**Greenfield project:**

```
> /bp:draft
What are you building?

> A REST API for task management. Users, projects, tasks with priorities
  and due dates, assignments. PostgreSQL.

Created 4 blueprints (22 requirements, 69 acceptance criteria)
Next: /bp:architect

> /bp:architect
Generated build site: 34 tasks, 5 tiers
Next: /bp:build

> /bp:build
Loop activated — 34 tasks, 20 max iterations.
...
All tasks done. Build passes. Tests pass.
BLUEPRINT COMPLETE — 34 tasks in 18 iterations.
```

**Existing codebase:**

```
> /bp:draft --from-code
Exploring codebase... Next.js 14, Prisma, NextAuth.
Created 6 blueprints — 4 requirements are gaps (not yet implemented).

> /bp:architect --filter collaboration
Generated build site: 8 tasks, 3 tiers

> /bp:build
Loop activated — 8 tasks.
...
BLUEPRINT COMPLETE — 8 tasks in 8 iterations.
```

See [example.md](example.md) for full annotated conversations.

---

## Parallel Execution

`/bp:build` automatically parallelizes. When multiple tasks are ready (no unmet dependencies), it groups them into a few coherent work packets based on shared files, subsystem, and task complexity, then runs those packets in parallel.

```
> /bp:build
═══ Wave 1 ═══
3 task(s) ready:
  T-001: Database schema (tier 0, deps: none)
  T-002: Auth middleware (tier 0, deps: none)
  T-003: Config loader (tier 0, deps: none)

Dispatching 2 grouped subagents...
All 3 tasks complete. Merging...

═══ Wave 2 ═══
2 task(s) ready:
  T-004: User endpoints (tier 1, deps: T-001, T-002)
  T-005: Health check (tier 1, deps: T-003)

Dispatching 2 grouped subagents...
All done.

═══ BUILD COMPLETE ═══
Waves: 2 | Tasks: 5/5
```

How it works:
- Reads the build site and computes the **frontier** — all tasks whose dependencies are complete
- Groups the ready frontier into coherent work packets before delegating
- Uses parallel subagents where file ownership and task size make that worthwhile
- After all complete, merges results and computes the next frontier
- Repeats wave-by-wave until all tasks are done — no manual intervention between tiers

Circuit breakers prevent infinite loops: 3 test failures → task marked BLOCKED, all tasks blocked → stop and report.

---

## Codex Adversarial Review

Blueprint uses [Codex](https://github.com/openai/codex) (OpenAI's coding agent) as an adversarial reviewer — a second model with a fundamentally different perspective that catches blind spots Claude cannot see in its own output. This dual-model approach operates at three levels:

### Design Challenge — catch spec flaws before building

After Claude drafts blueprints and the internal reviewer approves them, the entire blueprint set is sent to Codex for a **design challenge** — an adversarial review focused exclusively on architecture-level concerns:

```
  Claude drafts            Blueprint           Codex challenges         User reviews
  blueprints ──────► reviewer approves ──────► the design ──────► blueprints + findings
                                                    │
                                          Checks:   │
                                          • Domain decomposition quality
                                          • Missing requirements
                                          • Ambiguous acceptance criteria
                                          • Implicit assumptions
                                          • Cross-domain coherence
```

Codex returns structured findings categorized as **critical** (must fix before building) or **advisory** (worth considering). Critical findings trigger an auto-fix loop — Claude addresses them, Codex re-challenges, up to 2 cycles. Advisory findings are presented alongside blueprints at the user review gate.

The design challenge is purpose-built to prohibit implementation feedback. No framework suggestions, no file path opinions — only design-level concerns that would cause real problems during the build phase.

### Tier Gate — catch code defects between build tiers

During `/bp:build`, every completed tier triggers a Codex adversarial code review before advancing:

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

The **severity-based gate** classifies findings by impact:

| Severity | Behavior |
|----------|----------|
| P0 (critical) | Blocks tier advancement. Fix task generated automatically. |
| P1 (high) | Blocks tier advancement. Fix task generated automatically. |
| P2 (medium) | Deferred. Logged but does not block. |
| P3 (low) | Deferred. Logged but does not block. |

Gate modes are configurable: `severity` (default — P0/P1 block), `strict` (all findings block), `permissive` (nothing blocks), or `off`.

The review-fix cycle runs up to 2 iterations per tier. After that, the build advances with a warning — the system never deadlocks.

### Speculative Review — eliminate gate latency

By default, Blueprint runs the Codex review of the *previous* tier in the background while Claude builds the *current* tier:

```
  Tier 0 complete ───────────────────────────────► Tier 1 complete
       │                                                │
       └── Codex reviews Tier 0 (background) ──────────►│
                                                        │
                              Results ready ◄───────────┘
                              before gate runs
```

When the current tier finishes and the gate checks for the previous tier's review, the results are already available — cutting tier gate latency to near-zero. If the background review isn't done yet, the system waits (with a configurable timeout) and falls back to synchronous review if needed.

### Command Safety Gate

A PreToolUse hook intercepts every Bash command before execution and classifies its safety:

```
  Agent runs bash command
       │
       ▼
  Fast-path check ──► allowlist (50+ safe commands) → approve
       │           └► blocklist (rm -rf, force push, DROP TABLE, ...) → block
       │
       ▼ (ambiguous)
  Codex classifies ──► safe → approve
       │            └► warn → approve + log
       │            └► block → prevent execution
       │
       ▼ (cached)
  Verdict cache ──► normalized pattern match → reuse verdict
```

The gate integrates with Claude Code's permission system — commands already allowed or blocked in settings bypass the gate entirely. Verdicts are cached by normalized command pattern within the session to avoid redundant API calls. When Codex is unavailable, the gate falls back to static rules only — it never blocks a command solely because the classifier is unreachable.

### Graceful degradation

All Codex features are **additive**. When Codex is not installed:

- Design challenge is skipped — the internal blueprint reviewer still runs
- Tier gate is skipped — the build loop proceeds without review pauses
- Command gate falls back to static allowlist/blocklist only
- A one-time install nudge appears: `Tip: Install Codex for adversarial code review`

Blueprint works the same as before. Codex makes it harder to ship bad blueprints and bad code.

### Configuration

Blueprint settings can live in two places:

- User default: `~/.blueprint/config`
- Project override: `.blueprint/config`

Precedence is: project override > user default > built-in default.

| Setting | Values | Default | Purpose |
|---------|--------|---------|---------|
| `bp_model_preset` | `expensive` `quality` `balanced` `fast` | `quality` | Resolve `reasoning`, `execution`, and `exploration` models for Blueprint commands |
| `codex_review` | `auto` `off` | `auto` | Enable/disable Codex reviews |
| `codex_model` | model string | (Codex default) | Model for Codex calls |
| `tier_gate_mode` | `severity` `strict` `permissive` `off` | `severity` | How findings gate tier advancement |
| `command_gate` | `all` `interactive` `off` | `all` | Which sessions get command gating |
| `command_gate_timeout` | milliseconds | `3000` | Timeout for Codex safety classification |
| `speculative_review` | `on` `off` | `on` | Background review of previous tier |
| `speculative_review_timeout` | seconds | `300` | Max wait for speculative results |

Built-in model presets:

| Preset | Reasoning | Execution | Exploration |
|--------|-----------|-----------|-------------|
| `expensive` | `opus` | `opus` | `opus` |
| `quality` | `opus` | `opus` | `sonnet` |
| `balanced` | `opus` | `sonnet` | `haiku` |
| `fast` | `sonnet` | `sonnet` | `haiku` |

Use `/bp:config` to inspect or change the active preset.

Examples:

```bash
/bp:config
/bp:config list
/bp:config preset balanced
/bp:config preset fast --global
```

---

## Commands

### Claude Code slash commands

| Command | Phase | Description |
|---------|-------|-------------|
| `/bp:research` | Research | Deep multi-agent research — codebase + web, produces research brief |
| `/bp:draft` | Draft | Decompose requirements into domain blueprints (offers research if warranted) |
| `/bp:architect` | Architect | Generate a tiered build site from blueprints |
| `/bp:build` | Build | Auto-parallel build — dispatches independent tasks concurrently, progresses through tiers autonomously |
| `/bp:inspect` | Inspect | Gap analysis + peer review against blueprints |
| `/bp:config` | — | Show or update the active Blueprint execution preset |
| `/bp:codex-review` | — | Run standalone Codex adversarial review on current diff |
| `/bp:progress` | — | Check build site progress |
| `/bp:gap-analysis` | — | Compare built vs. intended |
| `/bp:revise` | — | Trace manual fixes back into blueprints |
| `/bp:help` | — | Show usage guide |

### CLI commands

| Command | Description |
|---------|-------------|
| `blueprint version` | Print version |

---

## File Structure

```
context/
├── blueprints/               # Domain blueprints (persist across cycles)
│   ├── blueprint-overview.md
│   └── blueprint-{domain}.md
├── sites/                    # Build sites (one per plan)
│   ├── build-site-*.md
│   └── archive/
├── impl/                     # Implementation tracking
│   ├── impl-{domain}.md
│   ├── impl-review-findings.md   # Codex review findings ledger
│   ├── impl-speculative-log.md   # Speculative review timing data
│   ├── loop-log.md
│   └── archive/
└── refs/                     # Reference materials (PRDs, API docs)
    ├── research-brief-{topic}.md   # Synthesized research brief
    └── research-{topic}/           # Raw findings + findings board

scripts/
├── bp-config.sh              # Canonical Blueprint config + model preset resolver
├── codex-detect.sh           # Codex binary and plugin detection
├── codex-config.sh           # Backward-compatible wrapper for bp-config.sh
├── codex-review.sh           # Adversarial code review invocation
├── codex-findings.sh         # Structured finding management
├── codex-gate.sh             # Severity-based tier gating + fix cycle
├── codex-design-challenge.sh # Design challenge for blueprint drafts
├── codex-speculative.sh      # Background speculative review pipeline
└── command-gate.sh           # PreToolUse command safety gate
```

---

## Methodology

Blueprint is built on a simple observation: LLMs are non-deterministic, but software engineering doesn't have to be. By applying the **scientific method** — hypothesize, test, observe, refine — we extract reliable outcomes from a stochastic process.

| Concept | Role |
|---------|------|
| **Blueprints** | The hypothesis — what you expect the software to do |
| **Validation gates** | Controlled conditions — build, tests, acceptance criteria |
| **Convergence loops** | Repeated trials — iterate until stable |
| **Implementation tracking** | Lab notebook — what was tried, what worked, what failed |
| **Revision** | Update the hypothesis — trace bugs back to blueprints |

The plugin ships with 8 specialized agents, a multi-agent research system, and 13 deep-dive skills covering the full methodology. When Codex is installed, the system operates as a **dual-model architecture** — Claude builds and Codex reviews — catching classes of errors that single-model self-review cannot detect.

<details>
<summary><strong>View all skills</strong></summary>

- **[Blueprint Writing](skills/blueprint-writing)** — how to write blueprints agents can consume
- **[Convergence Monitoring](skills/convergence-monitoring)** — detecting when iterations plateau
- **[Peer Review](skills/peer-review)** — six modes for cross-model review
- **[Validation-First Design](skills/validation-first)** — every requirement must be verifiable
- **[Context Architecture](skills/context-architecture)** — progressive disclosure for agent context
- **[Revision](skills/revision)** — tracing bugs upstream to blueprints
- **[Brownfield Adoption](skills/brownfield-adoption)** — adding Blueprint to an existing codebase
- **[Speculative Pipeline](skills/speculative-pipeline)** — overlapping phases for faster builds
- **[Prompt Pipeline](skills/prompt-pipeline)** — designing the prompts that drive each phase
- **[Implementation Tracking](skills/impl-tracking)** — living records of build progress
- **[Documentation Inversion](skills/documentation-inversion)** — docs for agents, not just humans
- **[Peer Review Loop](skills/peer-review-loop)** — combining Ralph Loop with cross-model review
- **[Core Methodology](skills/methodology)** — the full DABI lifecycle

</details>

---

## Why "Blueprint"

Most AI coding tools treat the agent as a black box — you prompt, it generates, you hope. Blueprint inverts this. **The specification is the product. The code is a derivative.** When the spec is clear, the code follows. When the code is wrong, the spec tells you why.

This matters because AI agents are getting better every month, but the fundamental problem remains: without a specification, there's nothing to validate against. Blueprint gives every agent — current and future — a contract to build from and a standard to meet.

With Codex adversarial review, Blueprint goes further: a second model with different training and different blind spots reviews both the specification and the implementation. Two models disagreeing is a signal. Two models agreeing is confidence.

---

## License

MIT
