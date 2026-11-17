# CHANGELOG

## v4.1.0 — the full loop

Additive. Backward compatible with v4.0.0 — the three-command core is
unchanged; `§R` is optional; existing `SPEC.md` files still parse. Every
change below traces to a documented pain point or a research finding, not
a hunch.

### added — four reach-for verbs

The core loop stays `spec → build → check`. Four new verbs join it, each
opt-in and right-sized — you reach for them only when the change earns it:

- **`/ck:grill`** — calibrated interrogation of a fuzzy idea into a sharp
  `§G`/`§C`, one question at a time, before a spec exists.
- **`/ck:research`** — external knowledge into the new `§R` log; every
  finding cites a source, unverified ones flagged, never written as fact.
- **`/ck:review`** — adversarial senior review of the spec *before* build:
  refutes rather than rubber-stamps, hardens `§V`, ends in a go/no-go gate.
- **`/ck:deepen`** — spare-budget design pass. Picks the one shallowest
  module, proposes a deeper shape, holds behavior constant (tests green
  before and after).

### added — format

- `§R RESEARCH` — optional pipe-table log of external knowledge (`id|topic|finding|src`).
- **Sectioned ownership** — each verb writes only the sections it owns; no
  verb rewrites a foreign section. `spec` remains the sole general mutator.
- **Right-size** rule — ceremony scales to blast radius, never to ego.

### changed

- `build` now names the *exact* test that proves each `§V` it touches (a
  verification contract) instead of "add tests" — "do TDD" alone backfires.
- `build` reads `§R` so it grounds in researched facts, not re-derivation.
- `check` reframed as the drift detector: run after each build, before each ship.
- skill descriptions kept ultra-tight.

### why — pain points → changes (sourced)

| pain point (source) | change |
|---|---|
| Token / context tax — spec-kit loads ~18.6k tokens every session (spec-kit #1401); BMAD burns 80–100k/step (#1188) | caveman descriptions keep cavekit's nine-skill set short |
| Specs drift silently with no detector (OpenSpec #1212; spec-kit #1686) | `check` reframed as the drift detector, run every build |
| Ceremony overkill — 10–15× overhead, "sledgehammer for a nut" (BMAD #2003; HN 45610996) | **right-size** rule; core stays 3 commands; verbs are opt-in |
| Agents ignore the spec / mark done without doing (spec-kit #230; BMAD #446) | `build` verification contract names which test proves each `§V` |
| "Process without library context = organized hallucinations" (Tessl) | `/research` + durable `§R` external-knowledge log |
| No tool adversarially reviews the *plan* before build (competitor scan) | `/review` — separate skeptic anchored to an external oracle |
| Tools overwrite/delete spec files (Kiro #5239; Conductor) | **sectioned ownership** — no verb rewrites a section it does not own |

### why — research backing

- Spec as durable external memory across context resets — Anthropic,
  *Effective Context Engineering* (2025); the core justification for SDD.
- Plan in a separate phase — ADaPT (NAACL 2024).
- Verification contract names *which* tests, not "do TDD" — TDAD (2026).
- Critique must be external + adversarial, not introspective — LLMs cannot
  self-correct alone (Huang et al., ICLR 2024); separate-critic debate works
  (Du et al., ICML 2024).
- Gate effort by difficulty — Self-Critique Paradox (Snorkel, 2025);
  right-size follows directly.
- Deep modules — Ousterhout, *A Philosophy of Software Design* (`/deepen`).

---

## v4.0.0 — the rewrite

Full rewrite. Not backward compatible with v3.x. Different shape, same name.

### philosophy

Kept only what earned its tokens:

- `SPEC.md` — durable, addressable, caveman-encoded
- three commands — `/ck:spec`, `/ck:build`, `/ck:check`
- two skills — `caveman` encoding, `backprop` protocol

### added

- single `SPEC.md` format with six addressable sections (§G §C §I §V §T §B)
- pipe-table encoding for §T (tasks) and §B (bugs)
- caveman symbol set (→ ∴ ∀ ∃ ! ? ⊥ ≠ ∈ ∉ ≤ ≥ & |) as default for spec writes
- bug → §B → §V backprop reflex wired into `/ck:build` failure path
- `/ck:spec from-code` — distill spec from existing codebase
- `/ck:check` — read-only drift report (replaces five v3 review flavors)
- `npx skills add JuliusBrussee/cavekit` one-line install path (commands + skills)

### removed (relative to v3.1.0)

- 13 of 16 commands (sketch/map/make/ship/review/revise/status/init/config/resume/help/design/research/team/make-parallel)
- all 12 named sub-agents
- 19 of 21 skills
- Go binary and source (`cmd/`, `internal/`, `bin/`, `cavekit` executable)
- shell hooks (`hooks/`, `scripts/cavekit-launch-session.sh`, stop-hook state machine)
- TS tooling (`scripts/cavekit-picker.ts`, `scripts/cavekit-router.cjs`)
- Codex peer-review bridge (`.codex-plugin/`)
- `context/kits/`, `context/plans/`, `context/impl/`, `context/refs/` directories
- autonomous loop, per-task budgets, model-tier routing
- design-system `DESIGN.md` workflow
- knowledge-graph `graphify-out/` integration
- parallel wave execution and team mode
- `install.sh` (216 lines → 0)

### changed

- caveman was opt-in for inter-agent chatter in v3; default for spec writes in v4
- version: 3.1.0 → 4.0.0 (major rewrite, semver respected)
- README, plugin metadata, marketplace entry

### migration

See [`UPGRADE.md`](./UPGRADE.md). No automated migrator — the v3 kit
shape does not map cleanly to v4's single file. Recommended path: run
`/ck:spec from-code` on your existing v3 project to distill a v4 spec
from your built code.

### v3 reachability

v3 is frozen at tag `v3.1.0`. Stays installable and documented. Fixes
only for critical bugs; no new features.

---

## v3.1.0 and prior

See git log before the `v4.0.0` commit, or check out `v3.1.0`:

```bash
git checkout v3.1.0
```
