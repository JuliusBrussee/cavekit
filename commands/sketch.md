---
name: ck-sketch
description: "Write kits: decompose what you're building into domains with testable requirements"
argument-hint: "[REFS_PATH | --from-code] [--filter PATTERN]"
---

> **Note:** `/bp:draft`, `/ck:draft`, `/bp:sketch` are deprecated aliases. Use `/ck:sketch` instead.

# Cavekit Draft — Write Kits

This is the first phase of Cavekit. You are writing implementation-agnostic kits that define WHAT to build through collaborative design with the user.

**HARD GATE:** Do NOT generate cavekit files until you have presented the design and the user has approved it. This applies to EVERY project regardless of perceived simplicity. A todo app, a config change, a single-domain project — all of them go through the design process. The design can be short for simple projects, but you MUST present it and get approval.

## Step 0: Resolve Execution Profile

Before doing any substantive work:

1. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" summary` and print that exact line once.
2. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" model exploration` and store it as `EXPLORATION_MODEL`.
3. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" model reasoning` and store it as `REASONING_MODEL`.
4. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" caveman-active draft` and treat the result as `CAVEMAN_ACTIVE` (true/false). Draft phase is NOT in the default caveman_phases, so this will typically be false. If true, apply caveman-speak to research agent prompts and internal summaries only — never to kit content, user-facing design proposals, or questions.

Keep the user Q&A in the parent thread. Use `EXPLORATION_MODEL` for helper exploration/research and `REASONING_MODEL` for cavekit generation and review.

## Determine Mode

Parse `$ARGUMENTS`:
- If `--from-code` → **Brownfield mode** (reverse-engineer kits from existing code)
- If a path is given → **Refs mode** (generate kits from reference materials at that path)
- If no arguments → **Interactive mode** (collaborative design with the user)

## Step 1: Ensure Directories Exist

Create these if missing (no separate init needed):
- `context/kits/`
- `context/plans/`
- `context/impl/`
- `context/impl/archive/`
- `context/refs/`

## Step 2: Explore Project Context

Before asking ANY questions, understand what already exists:

1. Check for existing `context/kits/` — are there prior kits?
2. Read project docs, README, CLAUDE.md if present
3. Check recent git commits to understand current momentum
4. Scan the codebase structure (directory layout, key files)
5. Check `context/refs/` for reference materials already provided
6. Check for existing `DESIGN.md` at project root or in `context/designs/` — if present, this constrains visual design decisions for any UI-related kits

This gives you grounding before engaging the user. Do NOT skip this even if the user has already described what they want.

If the repo scan is non-trivial, dispatch one helper exploration subagent with `model: "{EXPLORATION_MODEL}"` to gather the codebase/docs/git summary, then continue the design conversation in the parent thread.

## Step 3: Gather Input

### Interactive mode (no arguments)

This is a **collaborative design process**. Follow these steps in order:

#### 3a: Offer Visual Companion (if applicable)

If upcoming questions will involve visual content (UI layouts, architecture diagrams, data flow), offer the visual companion. This MUST be its own message — do not combine it with any other content:

> "Some of what we're working on might be easier to explain if I can show it to you in a web browser. I can put together mockups, diagrams, comparisons, and other visuals as we go. Want to try it? (Requires opening a local URL)"

Wait for the user's response. If they decline, proceed with text-only. If they accept, read `references/visual-companion.md` for the detailed guide. The server lives in `scripts/visual-companion/`.

If the project has a `DESIGN.md` at the root, mention it: "I see an existing design system (DESIGN.md). I'll use it as a visual constraint for any UI-related kits."

**Per-question decision:** Even after the user accepts, decide FOR EACH QUESTION whether to use the browser or the terminal. The test: **would the user understand this better by seeing it than reading it?**
- **Use the browser** for: mockups, wireframes, architecture diagrams, side-by-side visual comparisons
- **Use the terminal** for: requirements questions, conceptual choices, tradeoff lists, scope decisions

#### 3b: Scoping Questions

Ask **1–2 initial questions** to understand the project's purpose, scope, and technology landscape. These are high-level — enough to assess whether deep research is warranted.

- Prefer **multiple choice** when possible
- Focus on: **what** you're building, **who** it's for, **what technology** is involved
- Open-ended is fine when the topic needs exploration

**Scope check:** If the request describes multiple independent subsystems (e.g., "build a platform with chat, file storage, billing, and analytics"), flag this immediately. Help the user decompose into sub-projects first. Each sub-project gets its own draft cycle.

#### 3c: Check for Existing Research Brief

After scoping, check for an existing research brief:

1. Look in `context/refs/` for `research-brief-*.md` files relevant to this project
2. If a **fresh brief exists** (check the `Generated:` date in the brief):
   ```
   Found research brief: research-brief-{topic}.md ({age} ago). Use it? [Y/n/rerun]
   ```
   - **Y**: Load the brief, skip to 3e
   - **n**: Ignore the brief, skip to 3e
   - **rerun**: Run fresh research (3d)

3. If **no brief exists**, assess whether research is warranted based on the scoping conversation:

   **Research IS recommended when any of:**
   - The project involves technology you have low confidence about (novel frameworks, niche domains)
   - The project is brownfield and the codebase is medium-to-large
   - The user's description implies architectural decisions with multiple viable approaches
   - The domain has fast-moving best practices (security, AI/ML, web frameworks)

   **Research is NOT recommended when:**
   - The project is a small config change or bug fix
   - The technology stack is well-understood and you have high confidence
   - The user has already provided comprehensive reference materials in `context/refs/`
   - The user explicitly asked for a quick draft

4. If research is warranted, prompt:
   ```
   This project touches {topics} which would benefit from deep research —
   current best practices, library landscape, and {codebase analysis if brownfield}.
   This typically takes 30-60 seconds and produces a research brief.

   Run deep research? [Y/n]
   ```
   - **Y**: Proceed to 3d
   - **n**: Skip to 3e (no penalty, no nag)

#### 3d: Research Phase

If research was triggered in 3c, run the deep research pipeline. Generate a topic slug from the project description (kebab-case, 2-4 words).

**1. Assess project size** — count source files to determine codebase agent count:

| File count | Codebase agents |
|-----------|----------------|
| 0 (greenfield) | 0 |
| 1–20 | 1 |
| 21–200 | 2 |
| 201+ | 3–4 |

Use standard depth (3 web agents).

**2. Generate research plan:**
- Use fixed categories (codebase: architecture, patterns, dependencies, tests; web: library-landscape, best-practices, existing-art, pitfalls)
- Generate 2–6 project-specific sub-questions from the scoping conversation
- Assign sub-questions to the most relevant category agent

**3. Create research directory:**
- `context/refs/research-{topic}/findings-board.md` with initial empty structure

**4. Dispatch wave 1 in parallel** (all codebase agents + web agents for library-landscape and existing-art):

Each codebase agent (subagent_type: `Explore`, model: `EXPLORATION_MODEL`):
```
Agent(
  subagent_type: "Explore",
  model: "{EXPLORATION_MODEL}",
  description: "Research codebase {categories}",
  prompt: "ROLE: Codebase researcher ({categories}) for: {description}
  RESEARCH QUESTIONS: {assigned questions}
  Explore at 'medium' thoroughness. Answer with file:line evidence.
  OUTPUT: ## Agent: codebase-{categories} — findings with Evidence, Implication, Confidence"
)
```

Each wave 1 web agent (subagent_type: `general-purpose`, model: `EXPLORATION_MODEL`):
```
Agent(
  subagent_type: "general-purpose",
  model: "{EXPLORATION_MODEL}",
  description: "Research web {categories}",
  prompt: "ROLE: Web researcher ({categories}) for: {description}
  RESEARCH QUESTIONS: {assigned questions}
  Search web (2-3 searches per question). Prefer official docs, GitHub, engineering blogs.
  Findings from 2024+ only. Include source URLs and adoption metrics.
  OUTPUT: ## Agent: {categories} — findings with [source: URL] and Confidence"
)
```

**5. Collect wave 1** — write each agent's output to `context/refs/research-{topic}/raw-{slug}.md`, compile findings board.

**6. Dispatch wave 2 in parallel** (web agents for best-practices and pitfalls, with findings board in prompt):
```
Agent(
  subagent_type: "general-purpose",
  model: "{EXPLORATION_MODEL}",
  description: "Research web {categories}",
  prompt: "ROLE: Web researcher ({categories}) for: {description}
  SHARED FINDINGS: {findings-board.md contents}
  Skip answered questions. Go deeper on gaps. Flag contradictions with 'Conflict:' prefix.
  OUTPUT: ## Agent: {categories} — findings with [source: URL] and Confidence"
)
```

**7. Collect wave 2** — write raw findings, update findings board.

**8. Dispatch synthesizer:**
```
Agent(
  subagent_type: "general-purpose",
  model: "{REASONING_MODEL}",
  description: "Synthesize research findings",
  prompt: "ROLE: Research synthesizer for: {description}
  RAW FINDINGS: {all raw-*.md contents}
  FINDINGS BOARD: {findings-board.md contents}
  Cross-validate, resolve contradictions, filter tangential findings.
  OUTPUT: Follow research brief format — Summary, Key Findings (Architecture & Patterns,
  Library Landscape, Best Practices, Existing Art, Pitfalls), Contradictions & Open Questions,
  Codebase Context (if applicable), Implications for Design, Sources."
)
```

**9. Write brief** to `context/refs/research-brief-{topic}.md`.

**10. Present summary:**
```
Research complete ({N} agents). Key findings:
- {3-5 bullet points}

Open questions: {list, or "none"}

Full brief: context/refs/research-brief-{topic}.md
```

#### 3e: Clarifying Questions — One at a Time

Continue asking questions **one at a time** to fill in the remaining design picture. Do NOT dump multiple questions in a single message.

- Prefer **multiple choice** when possible — easier to answer than open-ended
- Focus on: **core requirements**, **scope boundaries**, **user journeys**, **constraints**, **success criteria**

**If research was done**, use findings to inform questions:
- Open questions from the research brief become clarifying questions
- Findings inform multiple-choice options (e.g., "Research shows OAuth 2.1 + PKCE is current standard, and your codebase already has passport.js — build on that or switch?")
- Don't re-ask what research already answered

Continue until you have a clear picture of:
- What the system must do (core requirements)
- What it must NOT do (scope boundaries)
- Who uses it and how (user journeys / entry points)
- What constraints exist (technical, organizational, timeline)
- What success looks like (how do we know it works?)

#### 3f: Propose 2-3 Approaches

Before settling on a design, propose **2-3 different domain decomposition approaches** with tradeoffs:

- Lead with your **recommended** approach and explain why
- Present alternatives with honest pros/cons
- Consider: coupling, complexity, parallelizability, testability
- Be ready to combine elements from different approaches based on feedback
- **If research was done**, back approaches with evidence (e.g., "Approach A uses chevrotain — research confirms 3x faster than nearley for grammars of this size")

Example:
> **Approach A (recommended): Three domains — Auth, API, Storage**
> Pros: Clean boundaries, independent testing. Cons: More cross-references.
>
> **Approach B: Two domains — Backend, Frontend**
> Pros: Simpler. Cons: Backend domain is too broad, harder to parallelize.
>
> **Approach C: Five fine-grained domains**
> Pros: Maximum parallelism. Cons: Over-decomposed, too many cross-references for this scope.

#### 3g: Present Design Incrementally

Once the user picks an approach, present the design **section by section**. Scale each section to its complexity — a few sentences if straightforward, more detail if nuanced.

For each proposed domain:
1. Present scope and boundaries
2. Present key requirements with acceptance criteria
3. Present cross-references and dependencies
4. Ask: "Does this look right so far?"

Only move to the next domain when the current one is approved. Be ready to revise.

**Design for isolation:** Each domain should:
- Have one clear purpose
- Communicate through well-defined interfaces
- Be understandable and testable independently
- Be small enough to reason about in a single context window

**YAGNI ruthlessly:** Remove features the user didn't ask for. If you're tempted to add "nice to have" requirements, don't. Smaller kits are better kits.

### Refs mode (path given)

Read all files at the given path (or `context/refs/` if the path is a directory). Catalog what you find: PRDs, API docs, design kits, research, architecture docs. Use these as the source of truth for cavekit generation.

After reading, still present a brief design summary to the user before generating files:
- "Here's what I found and how I plan to decompose it into domains: ..."
- Get approval before proceeding to file generation.

### Brownfield mode (`--from-code`)

Explore the existing codebase:
1. Read directory structure to understand architecture
2. Identify logical domains from code organization
3. For each domain, identify: entry points, data models, external dependencies, existing tests
4. Treat existing code as the reference material

After exploration, present your proposed decomposition to the user before generating files.

## Step 4: Decompose into Domains

Analyze the input and decompose into logical domains. Each domain should be:
- **Cohesive** — covers one area of functionality
- **Loosely coupled** — minimal dependencies on other domains
- **Independently specifiable** — can be described without implementation details of other domains
- **Right-sized** — small enough to hold in context, large enough to be meaningful

## Step 5: Generate Kits

**Only after user approves the design**, generate cavekit files.

Do NOT perform the actual cavekit writing inline in the parent thread. Dispatch a `ck:drafter` subagent with `model: "{REASONING_MODEL}"` to write the files, then review the result in the parent thread.

For each domain, create `context/kits/cavekit-{domain}.md`:

```markdown
---
created: "{CURRENT_DATE_UTC}"
last_edited: "{CURRENT_DATE_UTC}"
---

# Cavekit: {Domain Name}

## Scope
{What this domain covers}

## Requirements

### R1: {Requirement Name}
**Description:** {What must be true}
**Acceptance Criteria:**
- [ ] {Testable criterion 1}
- [ ] {Testable criterion 2}
**Dependencies:** {Other kits/requirements this depends on, or "none"}

### R2: ...

## Out of Scope
{Explicit exclusions — what this domain does NOT cover}

## Cross-References
- See also: cavekit-{related-domain}.md

## Changelog
```

If `--filter` is set, only generate kits for domains matching the filter pattern.

### Quality Rules — These Are Non-Negotiable

- Every file MUST have YAML frontmatter with `created` and `last_edited` dates (ISO 8601 UTC)
- Kits are **implementation-agnostic** — describe WHAT, never HOW
- Every requirement MUST have testable acceptance criteria
- If a requirement cannot be automatically validated, flag it as needing human review
- Cross-reference kits where domains interact
- Explicitly state what is out of scope
- Use R-numbered requirements (R1, R2, R3...)
- **YAGNI** — do not add requirements the user did not ask for
- If research was done, kits may reference the brief as source material: "See `context/refs/research-brief-{topic}.md` for library evaluation"
- **Design system integration** — for kits containing UI requirements, acceptance criteria SHOULD reference DESIGN.md sections/tokens where applicable (e.g., "Button uses primary CTA styling from DESIGN.md Section 4"). This makes the visual contract explicit and inspectable. Do NOT duplicate DESIGN.md content — reference by section/token name only.

### Brownfield-Specific Rules

- Describe what the code DOES, not how it's implemented
- For each acceptance criterion, verify the existing code satisfies it
- If code does NOT satisfy a criterion, mark it as `[GAP]`
- Note source files that informed each cavekit in a Source Traceability section

## Step 6: Create cavekit-overview.md

```markdown
---
created: "{CURRENT_DATE_UTC}"
last_edited: "{CURRENT_DATE_UTC}"
---

# Cavekit Overview

## Project
{Project name and description}

## Domain Index
| Domain | Cavekit File | Requirements | Status | Description |
|--------|-----------|-------------|--------|-------------|
| {domain} | cavekit-{domain}.md | {count} | DRAFT | {one-line} |

## Cross-Reference Map
| Domain A | Interacts With | Interaction Type |
|----------|---------------|-----------------|
| {domain} | {other domain} | {data flow / dependency / event} |

## Dependency Graph
{Which domains must be implemented before others}
```

## Step 7: Validate

1. Verify every cross-reference points to an existing cavekit
2. Verify no domain is referenced but missing a cavekit
3. Verify the dependency graph has no circular dependencies
4. Verify acceptance criteria across kits are consistent (no contradictions)
5. Verify no implementation details leaked into kits (no framework names, file paths, API choices)
6. (Brownfield only) Verify acceptance criteria against existing code

## Step 8: Cavekit Review Loop

After writing kits, dispatch a **cavekit-reviewer** subagent to verify quality:

```
Agent tool (subagent_type: "ck:cavekit-reviewer", model: "{REASONING_MODEL}"):
  description: "Review cavekit documents"
  prompt: |
    You are a cavekit reviewer. Verify these kits are complete and ready for the Architect phase.

    **Kits to review:** context/kits/

    ## What to Check

    | Category | What to Look For |
    |----------|------------------|
    | Completeness | TODOs, placeholders, "TBD", incomplete sections, missing Out of Scope |
    | Consistency | Internal contradictions, conflicting requirements across domains |
    | Clarity | Requirements ambiguous enough to cause an agent to build the wrong thing |
    | Scope | Each domain focused enough — not covering multiple independent subsystems |
    | YAGNI | Unrequested features, over-engineering, gold-plating |
    | Implementation leakage | Framework names, file paths, API choices in requirements |
    | Cross-references | Missing bidirectional links, dangling references |
    | Acceptance criteria | Vague, untestable, or subjective criteria |

    ## Calibration

    Only flag issues that would cause real problems during the Architect phase.
    A missing section, a contradiction, or a requirement so ambiguous it could be
    interpreted two different ways — those are issues. Minor wording improvements
    and stylistic preferences are not.

    Approve unless there are serious gaps that would lead to a flawed build site.

    ## Output Format

    ## Cavekit Review

    **Status:** Approved | Issues Found

    **Issues (if any):**
    - [Cavekit X, RN]: [specific issue] - [why it matters]

    **Recommendations (advisory, do not block approval):**
    - [suggestions for improvement]
```

**Review loop rules:**
- If **Approved**: proceed to Step 9
- If **Issues Found**: fix the issues, re-dispatch the reviewer
- Maximum **3 iterations** — if still not passing, present remaining issues to the user for guidance

## Step 9: User Review Gate

After the review loop passes, ask the user to review the written kits:

> "Kits written and validated. Files are in `context/kits/`. Please review them and let me know if you want to make any changes before we move to the Architect phase."

Wait for the user's response. If they request changes, make them and re-run Step 8. Only proceed once the user approves.

## Step 10: Report and Transition

```markdown
## Draft Report

### Domains: {count}
### Requirements: {count}
### Acceptance Criteria: {count}

### Dependency Order
1. {domain} — no dependencies (implement first)
2. {domain} — depends on {domain}

### Gaps / Open Questions
- {anything that couldn't be fully specified}

### Next Step
Run `/ck:map` to generate the build site from these kits.
```

Present the report. When the user is ready, transition to `/ck:map`.

---

## Key Principles

- **One question at a time** — do not overwhelm with multiple questions
- **Multiple choice preferred** — easier to answer than open-ended when possible
- **YAGNI ruthlessly** — remove unnecessary features from all kits
- **Explore alternatives** — always propose 2-3 approaches before settling
- **Incremental validation** — present design section by section, get approval before moving on
- **Design for isolation** — each domain has one purpose, clear interfaces, independently testable
- **No cavekit generation before design approval** — the design conversation IS the value
