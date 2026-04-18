<p align="center">
  <img src="https://em-content.zobj.net/source/apple/391/rock_1faa8.png" width="120" />
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
  <a href="#the-loop">The loop</a> •
  <a href="#install">Install</a> •
  <a href="#your-first-run">Your first run</a> •
  <a href="#adding-to-an-existing-repo">Existing repo</a> •
  <a href="#commands">Commands</a> •
  <a href="#how-it-works">How it works</a> •
  <a href="example.md">Examples</a>
</p>

<p align="center">
  <strong>🪨 Caveman Ecosystem</strong> &nbsp;·&nbsp;
  <a href="https://github.com/JuliusBrussee/caveman">caveman</a> <em>talk less</em> &nbsp;·&nbsp;
  <a href="https://github.com/JuliusBrussee/cavemem">cavemem</a> <em>remember more</em> &nbsp;·&nbsp;
  <strong>cavekit</strong> <em>build better</em> <sub>(you are here)</sub>
</p>

---

A [Claude Code](https://docs.anthropic.com/en/docs/claude-code) plugin that turns natural language into **specs**, specs into **parallel build plans**, and build plans into **working software** — with an autonomous loop, validation gates, and optional cross-model review.

You describe what you want. Cavekit writes the contract. Agents build from the contract. Every line of code traces to a requirement. Every requirement has acceptance criteria.

---

## The Loop

```
/ck:sketch   →   /ck:map   →   /ck:make   →   /ck:check
 what to         how to        build it       verify
  build          build it                       it
```

Four commands. In that order. That is the whole main cycle.

- **`/ck:sketch`** — decompose your project into domains, write kits with R-numbered requirements and testable acceptance criteria.
- **`/ck:map`** — read kits, generate a tiered build site with a task dependency graph.
- **`/ck:make`** — autonomous parallel build loop. Ready tasks grouped into packets, validated, merged wave by wave. Runs until all tasks are done or a budget trips.
- **`/ck:check`** — gap analysis against kits + peer review of the code. Produces an APPROVE / REVISE / REJECT verdict and auto-amends kits when gaps appear.

One shortcut: **`/ck:ship "<description>"`** runs all four end-to-end with no user gates. For tiny features and throwaways — the guided path is better for anything non-trivial, because the design conversation is where the value is.

---

## Install

```bash
git clone https://github.com/JuliusBrussee/cavekit.git ~/.cavekit
cd ~/.cavekit && ./install.sh
```

Registers the plugin with Claude Code, syncs into the Codex marketplace, installs the `cavekit` CLI. Restart Claude Code after installing.

**Requires:** [Claude Code](https://docs.anthropic.com/en/docs/claude-code), git, macOS/Linux.

**Optional:** [Codex](https://github.com/openai/codex) (`npm install -g @openai/codex`) — adds adversarial review. Cavekit works without it.

---

## Your First Run

Greenfield. New repo. You want a task-management API.

```
> /ck:init
  Context hierarchy created. Capabilities detected.
  Next: /ck:sketch

> /ck:sketch
  What are you building?

> A REST API for task management. Users, projects, tasks with
  priorities and due dates. PostgreSQL.

  (design conversation — research if warranted, domain decomposition,
   acceptance criteria refinement)

  4 kits, 22 requirements, 69 acceptance criteria.
  Next: /ck:map

> /ck:map
  34 tasks across 5 tiers. Coverage: 69/69 criteria mapped.
  Next: /ck:make

> /ck:make
  Loop active — 34 tasks, 20 max iterations.
  Wave 1 (3 tasks), Wave 2 (4 tasks), …
  ALL TASKS DONE. Build passes. Tests pass.
  Next: /ck:check

> /ck:check
  Coverage: 100%. Verdict: APPROVE.
  CAVEKIT COMPLETE
```

That is the experience.

## Adding to an Existing Repo

Same loop, different first command.

```
> /ck:sketch --from-code
  Scanning repo… Next.js 14, Prisma, NextAuth.
  6 kits reverse-engineered. 4 requirements flagged as gaps (not yet implemented).
  Next: /ck:map --filter collaboration   (or whichever domain you're adding to)

> /ck:map --filter collaboration
  8 tasks, 3 tiers.
  Next: /ck:make

> /ck:make
  …

> /ck:check
  Verdict: APPROVE. Design-system compliance: 100%.
```

See [example.md](example.md) for fully annotated sessions.

---

## Commands

The main cycle is four commands. Everything else is optional.

### Core

| Command | Phase | What it does |
|---------|-------|--------------|
| `/ck:sketch` | Draft | Decompose into domains; write kits with R-numbered requirements and testable acceptance criteria |
| `/ck:map` | Architect | Generate a tiered build site (task dependency graph) from kits |
| `/ck:make` | Build | Autonomous parallel build loop; validates each task against its criteria |
| `/ck:check` | Inspect | Gap analysis + peer review; verdict APPROVE / REVISE / REJECT |
| `/ck:ship` | End-to-end | One-shot sketch → map → make → check, no user gates. Tiny features only. |

### Auxiliary

| Command | What it does |
|---------|--------------|
| `/ck:init` | Bootstrap `context/` and `.cavekit/` runtime state. `--tools-only` re-detects capabilities. |
| `/ck:design` | Create / import / update / audit `DESIGN.md` (9-section visual system) |
| `/ck:research` | Parallel multi-agent research → brief in `context/refs/` |
| `/ck:revise` | Trace manual fixes back into kits. `--trace` runs the single-failure backpropagation protocol (auto-invoked on test failure). |
| `/ck:review` | Branch review: kit compliance + code quality. `--mode gap`, `--codex`, `--tier`, `--strict` narrow scope. |
| `/ck:status` | Task frontier and runtime state. `--watch` tails the live dashboard. |
| `/ck:config` | Execution presets and runtime keys. `--global` writes to `~/.cavekit/config`. |
| `/ck:resume` | Recover an interrupted loop. |
| `/ck:help` | Command reference. |

Run `/ck:help` for flag-level detail on any command.

### CLI

| Command | What it does |
|---------|--------------|
| `cavekit monitor` | Interactive launcher — pick build sites, launch in tmux |
| `cavekit status` | Build site progress |
| `cavekit kill` | Stop all sessions, clean up worktrees |
| `cavekit version` | Print version |
| `cavekit reset` | Clear persisted state |

---

## How It Works

Four phases. Each one a slash command.

### 1. Sketch — define the what

Describe what you're building in natural language. Cavekit decomposes it into **domain kits** — structured documents with numbered requirements (R1, R2, …) and testable acceptance criteria. Stack-independent. Human-readable.

If Codex is installed, kits go through a [design challenge](#codex-review-modes) — adversarial review that catches decomposition flaws before any code is written.

Brownfield: `/ck:sketch --from-code` reverse-engineers kits from your code and flags gaps.

### 2. Map — plan the order

Reads every kit. Breaks requirements into tasks. Maps dependencies. Organizes into a **tiered build site** — a dependency graph where Tier 0 has no deps, Tier 1 depends only on Tier 0, and so on. Includes a Coverage Matrix mapping every acceptance criterion to its task(s). Nothing specified gets lost in translation.

### 3. Make — run the loop

Pre-flight coverage check validates all acceptance criteria are covered. Then the loop runs:

```
┌──────────────────────────────────────────────────────┐
│  Read build site → Find next unblocked task          │
│       ▼                                              │
│  Load kit + acceptance criteria                      │
│       ▼                                              │
│  Implement task                                      │
│       ▼                                              │
│  Validate (build + tests + acceptance criteria)      │
│       ▼                                              │
│  PASS → commit → mark done → next ──┐                │
│  FAIL → diagnose → fix → revalidate │                │
│  ◄──────────────────────────────────┘                │
│                                                      │
│  Loop until: all tasks done OR budget exhausted      │
└──────────────────────────────────────────────────────┘
```

`/ck:make` parallelizes automatically. Multiple ready tasks get grouped into coherent work packets and dispatched concurrently:

```
═══ Wave 1 ═══
3 task(s) ready:
  T-001: Database schema   (tier 0)
  T-002: Auth middleware   (tier 0)
  T-003: Config loader     (tier 0)

Dispatching 2 grouped subagents…
All 3 tasks complete. Merging…

═══ Wave 2 ═══
2 task(s) ready:
  T-004: User endpoints    (tier 1, deps: T-001, T-002)
  T-005: Health check      (tier 1, deps: T-003)
…
```

Circuit breakers prevent infinite loops: 3 test failures → task BLOCKED; all blocked → stop and report.

At every tier boundary, optional [Codex review](#codex-review-modes) gates advancement. P0/P1 findings must be fixed before the next tier starts. Speculative review (default) adds near-zero latency.

### 4. Check — verify the result

Gap analysis: built vs. specified. Peer review: bugs, security, missed requirements. Everything traced back to kit requirements. Verdict: APPROVE / REVISE / REJECT. Gaps feed back as remediation tasks.

---

## Design System (optional)

If the project has UI, run `/ck:design` first. It creates or imports `DESIGN.md` — a 9-section Google-Stitch-format design system. Every kit then references its design tokens; every UI task carries a Design Ref; every build result is audited for design violations during `/ck:check`.

```
/ck:design                       # interactive
/ck:design --import vercel       # start from a known system
/ck:design --from-site <url>     # extract tokens from a live site
/ck:design --audit               # gap-check an existing DESIGN.md
```

---

## Configuration

Settings live in two places:

| Location | Scope |
|----------|-------|
| `~/.cavekit/config` | User default |
| `.cavekit/config` | Project override (takes precedence) |

### Model Presets

| Preset | Reasoning | Execution | Exploration |
|--------|-----------|-----------|-------------|
| `expensive` | opus | opus | opus |
| `quality` | opus | opus | sonnet |
| `balanced` | opus | sonnet | haiku |
| `fast` | sonnet | sonnet | haiku |

```
/ck:config                       # show current
/ck:config preset balanced       # change preset for this repo
/ck:config preset fast --global  # change default for all repos
```

<details>
<summary><strong>All configuration keys</strong></summary>

| Setting | Values | Default | Purpose |
|---------|--------|---------|---------|
| `bp_model_preset` | `expensive` `quality` `balanced` `fast` | `quality` | Model selection |
| `codex_review` | `auto` `off` | `auto` | Enable/disable Codex reviews |
| `codex_model` | model string | (Codex default) | Model for Codex calls |
| `tier_gate_mode` | `severity` `strict` `permissive` `off` | `severity` | How findings gate tier advancement |
| `command_gate` | `all` `interactive` `off` | `all` | Command-gating scope |
| `command_gate_timeout` | ms | `3000` | Codex classification timeout |
| `speculative_review` | `on` `off` | `on` | Background review of previous tier |
| `speculative_review_timeout` | s | `300` | Max wait for speculative results |
| `caveman_mode` | `on` `off` | `on` | Token-compressed output (~75% savings) |
| `caveman_phases` | comma-separated | `build,inspect` | Which phases use caveman-speak |
| `session_budget` | tokens | `500000` | Loop token cap before auto-halt |
| `max_iterations` | integer | `60` | Stop-hook iteration cap |
| `task_budget_quick` | tokens | `8000` | Per-task budget for `depth: quick` |
| `task_budget_standard` | tokens | `20000` | Per-task budget for `depth: standard` |
| `task_budget_thorough` | tokens | `45000` | Per-task budget for `depth: thorough` |
| `auto_backprop` | `on` `off` | `on` | Trigger backpropagation on test failure |
| `tool_cache` | `on` `off` | `on` | Cache read-only tool results |
| `tool_cache_ttl_ms` | ms | `120000` | TTL for cached tool results |
| `test_filter` | `on` `off` | `on` | Condense test output around failures |
| `progress_tracker` | `on` `off` | `on` | Write `.cavekit/.progress.json` |
| `parallelism_max_agents` | integer | `3` | Max concurrent subagents per wave |
| `parallelism_max_per_repo` | integer | `2` | Max concurrent subagents writing the same repo |
| `model_routing` | `on` `off` | `on` | Score-based tier routing |
| `graphify_enabled` | `on` `off` | `off` | Use knowledge-graph queries |

</details>

---

<details>
<summary><strong>Codex review modes</strong></summary>

Cavekit uses [Codex](https://github.com/openai/codex) as an adversarial reviewer — a second model with different training and different blind spots. Three levels:

### Design Challenge — catch spec flaws before building

After kits are drafted and internally reviewed, the full set goes to Codex.

| Finding type | Behavior |
|--------------|----------|
| **Critical** | Must fix before building. Auto-fix loop, up to 2 cycles. |
| **Advisory** | Presented alongside kits at user review gate. |

Only design-level concerns. No implementation feedback.

### Tier Gate — catch code defects between tiers

Every completed tier triggers a Codex code review before advancing.

| Severity | Behavior |
|----------|----------|
| **P0** (critical) | Blocks advancement. Auto-generates fix task. |
| **P1** (high) | Blocks advancement. Auto-generates fix task. |
| **P2** (medium) | Logged, does not block. |
| **P3** (low) | Logged, does not block. |

Gate modes: `severity` (default — P0/P1 block), `strict` (all block), `permissive` (nothing blocks), `off`.

Fix cycle runs up to 2 iterations per tier. After that, advances with warning. Never deadlocks.

### Speculative Review — eliminate gate latency

Codex reviews the previous tier in the background while Claude builds the current tier. Results are ready when the gate checks. Near-zero latency. Falls back to synchronous if needed.

### Command Safety Gate

PreToolUse hook intercepts every Bash command. Fast-path allowlist (50+ safe commands) / blocklist (rm -rf, force push, DROP TABLE). Ambiguous commands → Codex classifies → safe / warn / block. Verdict cached per session. Falls back to static rules when Codex is unavailable — never blocks solely because classifier is unreachable.

### Graceful Degradation

Without Codex installed: design challenge skipped, tier gate skipped, command gate falls back to static allowlist. Cavekit works the same; Codex makes it harder to ship bad specs and bad code.

</details>

<details>
<summary><strong>Autonomous runtime internals</strong></summary>

`/ck:make` is an autonomous loop. A Claude Code **Stop hook** drives the session iteration by iteration until every task is complete or a budget trips a circuit breaker.

- **`hooks/stop-hook.sh`** — state-machine driver. Fires on every Stop event, routes the next prompt, returns `{"decision":"block"}` so the session continues.
- **`hooks/token-monitor.sh`** — PostToolUse budget guard. Warns at 80% of per-task budget, halts at 100%.
- **`hooks/tool-cache.js` / `tool-cache-store.js`** — 120s TTL cache for read-only commands (`git status`, `ls`, `Read`, `Grep`, `Glob`).
- **`hooks/test-output-filter.js`** — condenses test output around failures.
- **`hooks/auto-backprop.js`** — on test failure, writes a flag file; next iteration prepends a trace directive.
- **`hooks/progress-tracker.js`** — zero-stdout snapshot writer for `/ck:status --watch`.
- **`scripts/cavekit-tools.cjs`** — orchestration engine: state machine, heartbeat lock, token ledger, task registry, routing, capability discovery, checkpoints, artifact summaries.
- **`scripts/cavekit-router.cjs`** — model-tier router. Scores tasks across five axes, maps to haiku/sonnet/opus within each role's band, demotes under budget pressure.

Runtime state under `<project>/.cavekit/`:

```
.cavekit/
├── config.json
├── state.md
├── .loop.json
├── .loop.lock
├── token-ledger.json
├── task-status.json
├── capabilities.json
├── .progress.json
├── .auto-backprop-pending.json
├── history/backprop-log.md
└── tool-cache/
```

Agents end the loop cleanly by emitting `<promise>CAVEKIT COMPLETE</promise>` on its own line. Debug with `CAVEKIT_DEBUG=1`. Recover with `/ck:resume`.

### Per-phase runtime calls

| Phase | Runtime call |
|-------|--------------|
| `/ck:init` | `cavekit-tools init` + `discover`; seeds `.cavekit/` and `.gitignore` |
| `/ck:sketch` | dispatches `ck:complexity` per kit to auto-fill `complexity:` |
| `/ck:map` | writes `.cavekit/tasks.json` + `init-registry` + `cavekit-router` |
| `/ck:make` | `setup-build.sh` calls `setup-loop`; stop-hook drives waves |
| `/ck:check` | dispatches `ck:verifier` for goal-backward check |
| `/ck:review` | two-pass review; fix-cycle emits fix tasks back into the loop |
| `/ck:revise` | routes each manual fix through the single-failure trace |
| `/ck:status` | prints live runtime status; `--watch` tails snapshots |
| `/ck:resume` | steals stale locks, validates state, re-enters the loop |
| `/ck:config` | surfaces runtime keys alongside preset controls |

Opt in per repo with `/ck:init`. Commands fall back to the pre-3.0 path when `.cavekit/` is absent.

</details>

<details>
<summary><strong>File structure</strong></summary>

```
context/                       # Project artifacts (persist across cycles)
├── kits/
│   ├── cavekit-overview.md
│   └── cavekit-{domain}.md
├── designs/
│   ├── DESIGN.md
│   └── design-changelog.md
├── plans/
│   └── build-site.md
├── impl/
│   ├── impl-{domain}.md
│   ├── impl-review-findings.md
│   └── loop-log.md
└── refs/

.cavekit/                      # Runtime state (machine-managed)
├── config.json
├── state.md
├── .loop.json
├── .loop.lock
├── token-ledger.json
├── task-status.json
├── capabilities.json
├── .progress.json
├── .auto-backprop-pending.json
├── history/backprop-log.md
└── tool-cache/
```

</details>

<details>
<summary><strong>Skills</strong></summary>

| Skill | What it covers |
|-------|----------------|
| [Methodology](skills/methodology) | Core Hunt lifecycle |
| [Design System](skills/design-system) | Create and maintain DESIGN.md |
| [UI Craft](skills/ui-craft) | Component patterns, animation, accessibility, review checklist |
| [Cavekit Writing](skills/cavekit-writing) | Write kits agents can consume |
| [Peer Review](skills/peer-review) | Six review modes + Codex Loop Mode |
| [Validation-First Design](skills/validation-first) | Every requirement must be verifiable |
| [Context Architecture](skills/context-architecture) | Progressive disclosure for agent context |
| [Revision](skills/revision) | Trace bugs upstream to kits (includes automated backprop) |
| [Convergence Monitoring](skills/convergence-monitoring) | Detect when iterations plateau |
| [Impl Tracking](skills/impl-tracking) | Living records of build progress |
| [Brownfield Adoption](skills/brownfield-adoption) | Add Cavekit to existing codebases |
| [Speculative Pipeline](skills/speculative-pipeline) | Overlap phases for faster builds |
| [Prompt Pipeline](skills/prompt-pipeline) | Design the prompts driving each phase |
| [Documentation Inversion](skills/documentation-inversion) | Docs for agents, not just humans |
| [Karpathy Guardrails](skills/karpathy-guardrails) | Think-before-code, simplicity, surgical changes |
| [Autonomous Loop](skills/autonomous-loop) | State machine, sentinels, lock protocol |
| [Caveman](skills/caveman) | Token-compressed output (~75% savings) |

</details>

<details>
<summary><strong>Methodology</strong></summary>

Cavekit applies the **scientific method** to AI-generated code. LLMs are non-deterministic. Software engineering does not have to be.

| Concept | Role |
|---------|------|
| **Kits** | The hypothesis — what you expect the software to do |
| **Validation gates** | Controlled conditions — build, tests, acceptance criteria |
| **Convergence loops** | Repeated trials — iterate until stable |
| **Implementation tracking** | Lab notebook — what was tried, what worked, what failed |
| **Revision** | Update the hypothesis — trace bugs back to kits |

Ships with specialized agents (including **design-reviewer** for UI validation against DESIGN.md), a multi-agent research system, and 21 skills. With Codex, operates as a **dual-model architecture** — Claude builds, Codex reviews — catching errors single-model self-review cannot.

**The spec is the product. The code is a derivative.**

When the spec is clear, the code follows. When the code is wrong, the spec tells you why.

Two models disagreeing is a signal. Two models agreeing is confidence.

</details>

---

## Star This Repo

If cavekit save you mass debug time — leave star.

[![Star History Chart](https://api.star-history.com/svg?repos=JuliusBrussee/cavekit&type=Date)](https://star-history.com/#JuliusBrussee/cavekit&Date)

---

## 🪨 The Caveman Ecosystem

Three tools. One philosophy: **agent do more with less**.

| Repo | What | One-liner |
|------|------|-----------|
| [**caveman**](https://github.com/JuliusBrussee/caveman) | Output compression skill | *why use many token when few do trick* — ~75% fewer output tokens across Claude Code, Cursor, Gemini, Codex |
| [**cavemem**](https://github.com/JuliusBrussee/cavemem) | Cross-agent persistent memory | *why agent forget when agent can remember* — compressed SQLite + MCP, local by default |
| [**cavekit**](https://github.com/JuliusBrussee/cavekit) *(you are here)* | Spec-driven autonomous build loop | *why agent guess when agent can know* — natural language → kits → parallel build → verified |

They compose: **cavekit** orchestrates the build, **caveman** compresses what the agent *says* (bundled, on by default for build/inspect phases), **cavemem** compresses what the agent *remembers* across sessions. Install one, some, or all — each stands alone.

## Also by Julius Brussee

- **[Revu](https://github.com/JuliusBrussee/revu-swift)** — local-first macOS study app with FSRS spaced repetition. [revu.cards](https://revu.cards)

## License

MIT
