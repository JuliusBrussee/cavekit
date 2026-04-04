---
name: bp-quick
description: "Quick end-to-end: describe a feature, get it built — draft, architect, build, and inspect without stopping"
argument-hint: "<feature description> [--skip-inspect] [--peer-review] [--max-iterations N]"
allowed-tools: ["Bash(${CLAUDE_PLUGIN_ROOT}/scripts/setup-build.sh:*)", "Bash(${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh:*)", "Bash(git *)"]
---

# Blueprint Quick — End-to-End Feature Build

Run the full Blueprint pipeline (draft → architect → build → inspect) from a single feature description with no stops for user input. Draft and architect phases are streamlined — no interactive design conversation, no user gates between phases.

**When to use:** Small-to-medium features where you trust the agent's decomposition. For large or ambiguous projects, use `/bp:draft` interactively instead.

## Phase 0: Resolve Execution Profile

Before starting the pipeline:

1. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" summary` and print that exact line once.
2. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" model reasoning` and store it as `REASONING_MODEL`.
3. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" model execution` and store it as `EXECUTION_MODEL`.
4. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" model exploration` and store it as `EXPLORATION_MODEL`.

Use those resolved model strings explicitly in every delegated phase below. Do not rely on agent frontmatter model defaults.

## Parse Arguments

Extract from `$ARGUMENTS`:
- **Feature description** — everything that isn't a flag (required)
- `--skip-inspect` — skip the inspect phase after build
- `--peer-review` — pass through to build phase
- `--max-iterations N` — pass through to build phase

If no feature description is provided:
> Usage: `/bp:quick <describe what you want built>`
>
> Example: `/bp:quick add a REST API for user profiles with CRUD operations and JWT auth`

Stop and wait for user input.

---

## Phase 1: Quick Draft

Streamlined version of `/bp:draft` — no interactive Q&A, no approach proposals, no incremental presentation.

Do NOT run this phase inline in the parent thread. Dispatch a `bp:drafter` subagent with `model: "{REASONING_MODEL}"` and give it the quick-draft rules below. If that subagent needs helper exploration for repo scanning, it should dispatch those helpers with `model: "{EXPLORATION_MODEL}"`.

### 1a: Ensure Directories

Create if missing: `context/blueprints/`, `context/plans/`, `context/impl/`, `context/impl/archive/`, `context/refs/`

### 1b: Explore Context

Silently gather context (do NOT present findings to user):
1. Check for existing blueprints in `context/blueprints/`
2. Read README, CLAUDE.md if present
3. Scan codebase structure (directory layout, key files, package.json/Cargo.toml/etc.)
4. Check recent git commits (`git log --oneline -10`)
5. Check for `DESIGN.md` at project root — if present, it constrains UI decisions

### 1c: Decompose and Write Blueprints

Using the feature description + project context, directly:

1. **Decompose** into domains (prefer fewer — 1 domain is fine for small features)
2. **Write** `context/blueprints/blueprint-{domain}.md` files following the standard format:
   - YAML frontmatter with `created` and `last_edited`
   - R-numbered requirements with testable acceptance criteria
   - Out of Scope section
   - Cross-references if multiple domains
3. **Write** `context/blueprints/blueprint-overview.md`

**Quick Draft Rules:**
- YAGNI — only what the user described, nothing extra
- Prefer 1-2 domains over many — keep it tight
- Acceptance criteria must be testable but don't over-specify
- If DESIGN.md exists and the feature involves UI, reference design tokens in acceptance criteria
- Skip the visual companion, skip approach proposals
- Skip the blueprint-reviewer subagent loop — you validate inline
- Count your acceptance criteria — every one must be concrete enough that the architect phase can assign it to a task. If a criterion is vague ("works well", "good UX"), rewrite it to be testable before proceeding.
- Do a single self-check: no TODOs, no placeholders, no implementation details in requirements

### 1d: Report (brief)

```
--- Quick Draft ---
Domains: {count}
Requirements: {count}
Files: {list}
```

Proceed immediately — no user gate.

---

## Phase 2: Quick Architect

Streamlined version of `/bp:architect` — runs inline, no stopping.

Do NOT run this phase inline in the parent thread. Dispatch a `bp:architect` subagent with `model: "{REASONING_MODEL}"` and give it the quick-architect rules below.

### 2a: Read Blueprints

Read all blueprint files just written.

### 2b: Generate Build Site

1. Decompose requirements into T-numbered tasks
2. Organize into dependency tiers
3. Write `context/plans/build-site.md` with the standard format (tier tables + Mermaid graph)

**Quick Architect Rules:**
- Tasks should be M-sized, not XL
- Every requirement maps to at least one task
- Dependencies must be genuine blockers
- For UI tasks, include `Design Ref: DESIGN.md Section {N}` if DESIGN.md exists
- Skip asking user about existing sites — overwrite if one exists
- **Coverage gate:** After generating the build site, walk through every acceptance criterion in every blueprint requirement and confirm it maps to at least one task. If any criterion is uncovered, add a task for it before proceeding. This replaces the full blueprint-reviewer loop with a single self-check pass — fast but non-negotiable.
- Include the Coverage Matrix section in the build site output (same format as the full architect path)

### 2c: Report (brief)

```
--- Quick Architect ---
Tasks: {count}
Tiers: {count}
Tier 0 (parallel start): {count} tasks
```

Proceed immediately — no user gate.

---

## Phase 3: Build

Run the **full build phase** exactly as `/bp:build` defines it. This is NOT simplified — the build loop runs with all its rigor:

1. Execute `"${CLAUDE_PLUGIN_ROOT}/scripts/setup-build.sh"` with any passthrough flags (`--peer-review`, `--max-iterations`)
2. Run the execution loop: compute frontier → dispatch tasks → merge → track → repeat
3. Follow all circuit breakers (3 consecutive failures = BLOCKED, merge conflicts = stop)
4. All critical rules apply: form coherent work packets, delegate only the packets that benefit from parallel execution, merge after every wave

The build phase continues to use the explicit `EXECUTION_MODEL` resolved by `/bp:build`.

The build phase is where quality matters — no shortcuts here.

---

## Phase 4: Inspect (unless `--skip-inspect`)

If `--skip-inspect` was NOT passed, run the **full inspect phase** exactly as `/bp:inspect` defines it:

1. Dispatch a `bp:surveyor` subagent with `model: "{REASONING_MODEL}"` for the gap analysis
2. Dispatch a `bp:inspector` subagent with `model: "{REASONING_MODEL}"` for the peer review
3. Synthesize both results into the full inspect report with verdict (APPROVE / REVISE / REJECT)
4. Auto-revise blueprints and site if gaps found

---

## Final Report

After all phases complete, present:

```markdown
# Blueprint Quick — Complete

**Feature:** {original description}

## Pipeline Summary
| Phase | Status | Details |
|-------|--------|---------|
| Draft | Done | {n} domains, {n} requirements |
| Architect | Done | {n} tasks, {n} tiers |
| Build | Done | {n} waves, {n}/{n} tasks completed |
| Inspect | {Done/Skipped} | {verdict or "skipped"} |

## What Was Built
{2-3 sentence summary of what was implemented}

## Files Changed
{list key files created/modified}

## {If inspect ran and verdict is REVISE or REJECT}
### Remaining Work
{summary of gaps or findings}
Run `/bp:build` to address remaining tasks, or `/bp:inspect` for details.
```
