# cavekit v4: I wrote a framework, then I killed most of it

**TL;DR**: spec-driven development is still the right idea. my v3
implementation buried it under ceremony. v4 is the rewrite. three
commands, one file, no sub-agents. v3 stays reachable at tag `v3.1.0`
for anyone it still works for.

---

I built cavekit v3 to prove that spec-driven development could give AI
agents enough context to stop guessing. On that, it delivered. The part
I got wrong is everything else I wrapped around it.

## what was slop

Cavekit v3 had 16 slash commands. Twelve named sub-agents. Twenty-one
skills, several of them 20KB each. A Go binary. Shell hooks. A stop-hook
state machine. An autonomous execution loop. Per-task token budgets.
Model-tier routing. A Codex peer-review bridge. Knowledge-graph
integration. Design-system enforcement. Team mode with path-scoped
claims. Parallel wave execution.

Every feature had a reason when I added it. Stacked up, they became a
framework you had to learn before you could write a spec. Invocations
loaded thousands of tokens of meta-philosophy (karpathy-guardrails,
validation-first, methodology, context-architecture, convergence-monitoring)
that Claude already knows, re-read in prose every single time.

Parallel agents were the worst of it. They look impressive. In practice
they shatter flow, coordinate via tedious ledger files, and need a
separate review agent to merge their disagreements. I'd rather plan
once, execute serially, and ship.

Native Claude Code plan-then-execute is already good. It's the baseline
cavekit v3 was supposed to beat. After enough sessions I realized it
was often *losing* to the baseline — same work, more ceremony, more
tokens.

## what actually earned its keep

One thing. The spec as a durable artifact. That's the only thing SDD
gives you over plan-then-execute — a spec survives context resets,
diffs against code, absorbs bugs back into itself. Everything else in
cavekit existed to serve that, and most of it did more harm than good.

The other thing I actually liked: caveman compression. It worked. It
just wasn't applied where it mattered — I used it for inter-agent
chatter, not the artifacts that get loaded every invocation.

## v4

Rewrote from the ground up. Here's the whole surface:

- `SPEC.md` at repo root. Six sections (§G §C §I §V §T §B), addressable
  by section.id. Caveman-encoded by default, so fewer tokens than
  prose. §T (tasks) and §B (bugs) are pipe tables because that's the
  efficient shape for repeating records.
- Three commands:
  - `/ck:spec` — the sole mutator. Create, amend, or backprop a bug.
  - `/ck:build` — native plan-then-execute against the spec. On test
    failure, calls `/ck:spec bug:` before retrying. Backprop is a
    reflex, not a dashboard.
  - `/ck:check` — read-only drift report.
- Two skills: `caveman` (the encoding), `backprop` (the six-step
  bug→spec protocol).

That's it. No sub-agents. No autonomous loop. No parallel workers. No
hooks, no Go binary, no TypeScript helpers. No `install.sh`.

4,977 lines of commands and agents in v3 → 226 lines of commands in v4.
21 skills → 2. A 5MB binary → none.

## is this just native claude code with extra steps

Basically yes. That's the point. The one thing it adds is the spec
format and the backprop reflex. Those two earn their tokens. Nothing
else gets to.

If you look at v4 and think "this is just a markdown file and a
convention," you're right. That's the shape a working version of this
idea was supposed to have all along. I just had to build the overbuilt
version first to find it out.

## for existing users

v3 is frozen at tag `v3.1.0`. Still installable:

```bash
/plugin marketplace add juliusbrussee/cavekit@v3.1.0
/plugin install ck@cavekit
```

If your project has a live `context/kits/` investment, or you rely on
the autonomous loop, or your team has muscle memory on the Hunt
lifecycle: stay on v3.1.0. It's frozen, not abandoned. It still works.

If you want the distilled version, install the default branch. See
`UPGRADE.md` for the migration (short version: run `/ck:spec from-code`
on your existing project — your built code becomes the source of truth,
your old kits live in git history).

It is a two-way door. `SPEC.md` is plain markdown. Nothing traps you in
either direction.

## install

One line:

```bash
npx skills add JuliusBrussee/cavekit
```

Or via the Claude Code marketplace:

```bash
/plugin marketplace add juliusbrussee/cavekit
/plugin install ck@cavekit
```

## what I'm not doing

- Not claiming v4 is strictly better than v3 for everyone. Different
  shape, different tradeoffs.
- Not promising parallel/autonomous features will come back. If I need
  them again, they belong in a separate tool, not the spec framework.
- Not apologizing for the rewrite. The overbuilt version was how I
  learned which parts were real.

---

GitHub: https://github.com/juliusbrussee/cavekit
Default branch is v4. v3 lives at tag `v3.1.0`.
