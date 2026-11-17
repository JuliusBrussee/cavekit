> **Status: frozen (August 2026).** cavekit is no longer in active development.
> Everything below still installs and works, but expect no new features or
> fixes. Active development of the family lives in
> [caveman](https://github.com/JuliusBrussee/caveman) and
> [caveman-browse](https://github.com/JuliusBrussee/caveman-browse).

<h1 align="center">cavekit</h1>

<p align="center">
  <strong>compressed spec-driven development for claude code</strong><br/>
  <sub>one file · one loop · zero sub-agents</sub>
</p>

---

## what this is

Plan-then-execute forgets. SDD remembers — but most SDD frameworks bury
that value under agent swarms, dashboards, and ceremony that costs more
tokens than it saves.

Cavekit is the simplest full loop: **grill → spec → research → review →
build**, over one `SPEC.md` file, no sub-agents. Three commands you run
every time; four more you reach for only when the change earns it.

The spine is three properties that earn their tokens:

- **durable spec** — `SPEC.md` at repo root survives context resets. It is
  the agent's long-term memory: lose the window, reload the spec, keep going.
- **caveman encoding** — fewer tokens than prose. Symbols, fragments,
  pipe tables. That is the whole point.
- **backprop reflex** — every test failure becomes a `§B` entry; classes
  of bug become `§V` invariants the spec never forgets.

And one rule that keeps it from bloating into the frameworks it replaces:
**right-size**. A one-line fix is just `/build`. The full chain is for
genuinely uncertain or high-blast-radius work — never for a typo.

## commands

**the loop** — run these every time:

| cmd | job |
|---|---|
| `/ck:spec` | create / amend / backprop `SPEC.md`. Sole mutator. |
| `/ck:build` | native plan → execute against spec. Names which test proves each `§V`. Auto-backprops on failure. |
| `/ck:check` | read-only drift report. Lists §V / §I / §T violations. The drift detector. |

**reach for these** — only when the change earns the ceremony:

| cmd | job |
|---|---|
| `/ck:grill` | interrogate a fuzzy idea into a sharp `§G`/`§C`, one question at a time, before you spec. |
| `/ck:research` | gather external knowledge into `§R` so build grounds in facts, not hallucinations. Every finding cites a source. |
| `/ck:review` | adversarial senior review of the spec *before* build. Refutes, hardens `§V`, ends in a go/no-go gate. |
| `/ck:deepen` | spare-budget design pass — make one shallow module deep. Behavior held, tests green before & after. |

## install

One line, via the `skills` CLI:

```bash
npx skills add JuliusBrussee/cavekit
```

Installs nine skills into `~/.claude/skills/`: `spec`, `build`, `check`
(the loop), `grill`, `research`, `review`, `deepen` (reach-for), plus
`caveman` and `backprop` (the utilities). Claude activates each when its
trigger context matches — e.g. "write a spec for…" invokes `spec`, a fuzzy
idea invokes `grill`, a risky change before build invokes `review`. Claude
Code picks them up on next launch.

Or via the Claude Code marketplace (also adds the `/ck:spec`, `/ck:build`,
`/ck:check`, `/ck:grill`, `/ck:research`, `/ck:review`, `/ck:deepen` slash
commands):

```bash
/plugin marketplace add juliusbrussee/cavekit
/plugin install ck@cavekit
```

Or clone directly:

```bash
git clone https://github.com/juliusbrussee/cavekit.git ~/.claude/plugins/cavekit
```

## format

See [`FORMAT.md`](./FORMAT.md). Sections: §G goal, §C constraints, §I
interfaces, §R research (optional, pipe table), §V invariants, §T tasks
(pipe table), §B bugs (pipe table). Each verb owns specific sections —
no verb rewrites a section it does not own.

## files

```
FORMAT.md             spec schema + caveman encoding + sectioned ownership
commands/             seven thin slash-command entry points → the skills (loop + reach-for)
skills/spec           spec mutator — sole writer
skills/build          plan-execute, verification contract
skills/check          drift report
skills/grill          sharpen a fuzzy idea → §G/§C before spec
skills/research       external knowledge → §R, every finding sourced
skills/review         adversarial senior review of the spec → hardens §V
skills/deepen         spare-budget design pass — make one module deep
skills/caveman        encoding utility
skills/backprop       bug → spec protocol (six steps)
```

## non-goals

- no sub-agents. Main Claude does the work.
- no dashboards. `cat SPEC.md` is the dashboard.
- no parallel workers. One thread, one spec, one diff.
- no JSON / YAML spec bodies. Markdown + pipe tables.
- no hooks, no orchestration binaries, no TypeScript helpers.

---

## older cavekit (the Hunt lifecycle, v3.1.0 and earlier)

The previous generation is **not deprecated** — it is frozen at tag
[`v3.1.0`](https://github.com/juliusbrussee/cavekit/tree/v3.1.0) and
remains a fully working plugin.

**What it is**:

> Spec-driven AI development with an autonomous execution loop. Four-command
> Hunt lifecycle (`/ck:sketch` → `/ck:map` → `/ck:make` → `/ck:check`),
> plus `/ck:ship`, `/ck:review`, `/ck:revise`, `/ck:status`, `/ck:design`,
> `/ck:research`, `/ck:init`, `/ck:config`, `/ck:resume`, `/ck:help` — 16
> slash commands total. 12 named sub-agents. Per-task token budgets,
> stop-hook state machine, model-tier routing, auto-backpropagation from
> test failures, tool-result caching, Codex peer review, Karpathy
> behavioral guardrails, caveman token compression, knowledge-graph
> integration, and design-system enforcement. Parallel wave execution and
> team mode.

**Pick v3.1.0** if you want the full autonomous loop, parallel agents,
peer review, or design-system workflow. **Pick v4** if you want the
distilled loop — one spec, no orchestration, right-sized ceremony.

### install the older version

Marketplace:

```bash
/plugin marketplace add juliusbrussee/cavekit@v3.1.0
/plugin install ck@cavekit
```

Git:

```bash
git clone -b v3.1.0 https://github.com/juliusbrussee/cavekit.git
```

Full docs live at the tag — `git checkout v3.1.0` and read the README
there for command reference, skill catalog, and the Hunt lifecycle guide.

### choosing, or moving

See [`UPGRADE.md`](./UPGRADE.md). Honest framing:
- Stay on v3.1.0 if your project has active `context/kits/` investment.
- Move to v4 if you want fewer moving parts and smaller token bills.
- It is a **two-way door** — `SPEC.md` is plain markdown; nothing traps
  you in either direction.

## ecosystem

Cavekit is one rock in the caveman family:

| repo | what | status |
|---|---|---|
| [caveman](https://github.com/JuliusBrussee/caveman) | output compression skill + engine — *why use many token when few do trick* | live |
| [caveman-browse](https://github.com/JuliusBrussee/caveman-browse) | token-efficient browser automation | live |
| [cavegemma](https://github.com/JuliusBrussee/cavegemma) | Gemma 4 31B fine-tuned on caveman pairs — *why prompt every turn when weight remember* | labs |
| [cavemem](https://github.com/JuliusBrussee/cavemem) | cross-agent persistent memory — *why agent forget when agent can remember* | frozen |
| **cavekit** *(you here)* | spec-driven build loop — *why agent guess when agent can know* | frozen |

## philosophy

> The spec is the only artifact that earns its tokens. Everything else
> that costs tokens must either save more tokens later, or the user's
> attention, or it gets cut.

See [`CHANGELOG.md`](./CHANGELOG.md) for the full v3 → v4 break.

## license

MIT.
