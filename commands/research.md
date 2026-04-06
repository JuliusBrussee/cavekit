---
name: ck-research
description: "Deep research for grounding kits in evidence — current best practices, library landscape, and codebase analysis"
argument-hint: "<description> [--depth quick|standard|deep] [--web-only] [--codebase-only]"
---

> **Note:** `/bp:research` is deprecated and will be removed in a future version. Use `/ck:research` instead.

# Cavekit Research — Deep Multi-Agent Research

Run parallel multi-agent research to ground cavekit design in real evidence. Produces a named research brief in `context/refs/` that feeds into cavekit design.

**Core insight:** Kits designed without research are kits designed on vibes. Research turns the design conversation from "what do you want?" into "here's what we know — what do you want?"

## Parse Arguments

Extract from `$ARGUMENTS`:
- `description` — what you're building (everything that isn't a flag). **Required.**
- `--depth` — `quick`, `standard` (default), or `deep`
- `--web-only` — skip codebase research (useful for greenfield)
- `--codebase-only` — skip web research (air-gapped environments)

If no description is provided, ask: "What are you building? (One sentence is enough)" and wait.

Generate a **topic slug** from the description — kebab-case, 2-4 words. Example: "Build a Verse compiler targeting WASM" → `verse-compiler`. "Add real-time collaboration" → `realtime-collab`.

## Step 0: Resolve Execution Profile

Before dispatching any research agents:

1. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" summary` and print that exact line once.
2. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" model exploration` and treat the result as `EXPLORATION_MODEL`.
3. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" model reasoning` and treat the result as `REASONING_MODEL`.

Use `EXPLORATION_MODEL` for codebase/web researchers and `REASONING_MODEL` for the synthesizer.

## Step 1: Assess Project Size

Quickly count source files (exclude `node_modules`, `.git`, `dist`, `build`, `vendor`, `__pycache__`, `.next`, `.nuxt`):

```bash
find . -type f \( -name '*.ts' -o -name '*.tsx' -o -name '*.js' -o -name '*.jsx' -o -name '*.py' -o -name '*.rs' -o -name '*.go' -o -name '*.java' -o -name '*.rb' -o -name '*.swift' -o -name '*.kt' \) -not -path '*/node_modules/*' -not -path '*/.git/*' -not -path '*/dist/*' -not -path '*/build/*' -not -path '*/vendor/*' -not -path '*/__pycache__/*' -not -path '*/.next/*' | wc -l
```

Map to project category:

| File count | Category | Codebase agents (standard) |
|-----------|----------|---------------------------|
| 0 (or `--web-only`) | Greenfield | 0 |
| 1–20 | Small | 1 (covers all categories) |
| 21–200 | Medium | 2 |
| 201+ | Large | 3–4 |

### Depth Adjustments

| Depth | Codebase agents | Web agents | Searches per question |
|-------|----------------|------------|----------------------|
| `quick` | min(1, adaptive) | 2 | 1 |
| `standard` | adaptive (table above) | 3 | 2–3 |
| `deep` | max(4, or 0 if greenfield) | 4 | exhaustive, follow cross-references |

## Step 2: Generate Research Plan

### Fixed Categories

**Codebase categories** (skip if greenfield or `--web-only`):

| Category | What to investigate |
|----------|-------------------|
| Architecture | Directory structure, module boundaries, entry points, build system, framework versions |
| Patterns | Coding conventions, error handling, naming, state management, abstractions in use |
| Dependencies | Package manifests, external APIs, integration points, version constraints |
| Test infrastructure | Test framework, coverage setup, fixtures, CI config, tested vs. untested areas |

**Web categories** (skip if `--codebase-only`):

| Category | What to investigate |
|----------|-------------------|
| Library landscape | Libraries/tools for the core problem domain |
| Best practices | Current architectural patterns, security, performance |
| Existing art | How others solved similar problems, reference implementations |
| Pitfalls | Known issues, common mistakes, deprecated approaches |

### Project-Specific Sub-Questions

Based on the description, generate **2–6 additional sub-questions** that target the specific technology decisions and domain knowledge needed. Assign each to the most relevant category agent.

Examples:
- "Build a Verse compiler targeting WASM" → "How do existing effect systems handle type inference in compilation?" → assigned to existing-art
- "Add real-time collaboration to our editor" → "What CRDT libraries exist for TypeScript in 2026?" → assigned to library-landscape

### Agent Assignment

Consolidate categories based on available agent count:

| Depth | Web agents | Assignment |
|-------|-----------|------------|
| `quick` | 2 | A: library-landscape + existing-art, B: best-practices + pitfalls |
| `standard` | 3 | A: library-landscape, B: best-practices + existing-art, C: pitfalls |
| `deep` | 4 | One agent per category |

For codebase agents with count < 4, consolidate: 1 agent = all categories, 2 agents = architecture+dependencies / patterns+tests, 3 agents = architecture / patterns+dependencies / tests.

### Wave Assignment

- **Wave 1** (foundation): library-landscape, existing-art (+ all codebase agents)
- **Wave 2** (builds on wave 1): best-practices, pitfalls

For `quick` depth: all web agents run in wave 1, no wave 2.

## Step 3: Initialize Research Directory

Create the directory and initial files:

```
context/refs/research-{topic}/findings-board.md
```

Write the initial findings board:

```markdown
# Findings Board: {description}

> Shared coordination state for research agents.
> Later agents read this before searching to avoid duplicates and build on earlier work.

(No findings yet — agents will populate this.)
```

Also ensure `context/refs/` exists.

## Step 4: Dispatch Research Agents

### Wave 1: Codebase Agents + Foundation Web Agents

Dispatch all of these **in parallel** (multiple Agent calls in one message):

**Each codebase agent** (subagent_type: `Explore`, model: `EXPLORATION_MODEL`):

```
Agent(
  subagent_type: "Explore",
  model: "{EXPLORATION_MODEL}",
  description: "Research codebase {categories}",
  prompt: "ROLE: Codebase researcher ({categories}) for: {description}

RESEARCH QUESTIONS:
{list of fixed-category questions + project-specific sub-questions assigned to this agent}

INSTRUCTIONS:
1. Explore the codebase focused on: {categories}
2. Answer each research question with specific evidence (file paths, code snippets)
3. Note patterns, conventions, and constraints that affect design decisions
4. Flag anything surprising or concerning
5. Be thorough — 'medium' level exploration

OUTPUT FORMAT:
## Agent: codebase-{categories}

For each finding:
- Finding: {what you found}
- Evidence: {file:line references or code snippets}
- Implication: {how this affects the design}
- Confidence: HIGH/MEDIUM/LOW"
)
```

**Each wave 1 web agent** (subagent_type: `general-purpose`, model: `EXPLORATION_MODEL`):

```
Agent(
  subagent_type: "general-purpose",
  model: "{EXPLORATION_MODEL}",
  description: "Research web {categories}",
  prompt: "ROLE: Web researcher ({categories}) for: {description}

RESEARCH QUESTIONS:
{list of fixed-category questions + project-specific sub-questions assigned to this agent}

INSTRUCTIONS:
1. Search the web for each research question ({searches_per_question} searches per question)
2. Fetch and read the most relevant results (prefer official docs, not summaries)
3. Extract specific, actionable findings — not vague advice
4. Flag contradictions between sources

QUALITY RULES:
- Prefer official docs, GitHub repos with stars, and engineering blogs over SEO content farms
- Include publication dates — findings from 2024+ only unless it's foundational knowledge
- Note when a library/practice is deprecated or superseded
- Always include source URLs
- If a library comparison exists, include adoption metrics (npm downloads, GitHub stars)

OUTPUT FORMAT:
## Agent: {categories}
- Found: {specific finding} [source: URL]
- Found: {specific finding} [source: URL]
- Confidence: HIGH/MEDIUM/LOW

Repeat for each research question."
)
```

### Collect Wave 1 Results

After all wave 1 agents complete:

1. Write each agent's output to `context/refs/research-{topic}/raw-{category-slug}.md`
2. Compile all findings into the findings board — append each agent's output under its agent header
3. Write the updated `context/refs/research-{topic}/findings-board.md`

### Wave 2: Dependent Web Agents

Skip this step for `quick` depth.

Dispatch remaining web agents **in parallel** (subagent_type: `general-purpose`, model: `EXPLORATION_MODEL`):

```
Agent(
  subagent_type: "general-purpose",
  model: "{EXPLORATION_MODEL}",
  description: "Research web {categories}",
  prompt: "ROLE: Web researcher ({categories}) for: {description}

RESEARCH QUESTIONS:
{list of fixed-category questions + project-specific sub-questions assigned to this agent}

SHARED FINDINGS FROM EARLIER AGENTS (read first — skip questions already answered, go deeper on gaps):
---
{full contents of findings-board.md}
---

INSTRUCTIONS:
1. Read the shared findings above carefully
2. Skip questions already well-answered — go deeper on gaps and unanswered questions
3. Build on earlier findings — add nuance, find counterpoints, verify claims
4. Search the web ({searches_per_question} searches per question)
5. Flag contradictions with earlier agents' findings using 'Conflict:' prefix

QUALITY RULES:
- Prefer official docs, GitHub repos with stars, and engineering blogs over SEO content farms
- Include publication dates — findings from 2024+ only unless foundational
- Note when a library/practice is deprecated or superseded
- Always include source URLs

OUTPUT FORMAT:
## Agent: {categories}
- Found: {specific finding} [source: URL]
- Conflict: {if contradicts an earlier finding — explain both sides}
- Confidence: HIGH/MEDIUM/LOW"
)
```

### Collect Wave 2 Results

1. Write each agent's output to `context/refs/research-{topic}/raw-{category-slug}.md`
2. Append to the findings board

## Step 5: Two-Pass Synthesis

### Pass 1: Synthesizer Agent

Read all raw findings files and the findings board. Dispatch the synthesizer:

```
Agent(
  subagent_type: "general-purpose",
  model: "{REASONING_MODEL}",
  description: "Synthesize research findings",
  prompt: "ROLE: Research synthesizer for: {description}

You have raw findings from {N} research agents. Produce a single coherent research brief.

RAW FINDINGS:
---
{concatenated contents of all raw-*.md files from context/refs/research-{topic}/}
---

FINDINGS BOARD:
---
{contents of findings-board.md}
---

INSTRUCTIONS:
1. Cross-validate: findings confirmed by multiple agents independently = HIGH confidence
2. Resolve contradictions: when agents disagree, read both sides carefully. Pick one with reasoning, or preserve both if genuinely debatable
3. Filter: remove tangential findings that don't affect design decisions
4. Structure: organize into the research brief format below
5. Flag unknowns: areas where research was inconclusive or sources disagreed → 'Open Questions'
6. Extract implications: what design decisions does this research inform?

OUTPUT — follow this format exactly:

# Research Brief: {topic title}

**Generated:** {current date}
**Agents:** {N} codebase, {N} web
**Sources consulted:** {count unique URLs/repos referenced}

## Summary
{2-3 sentence executive summary of key findings that affect design decisions}

## Key Findings

### Architecture & Patterns
{only if codebase research was done}
- {finding} [confidence: HIGH/MEDIUM/LOW] [sources: N]

### Library Landscape
- Recommended: {library X} — {why, with metrics if available} [confidence: HIGH]
- Alternative: {library Y} — {tradeoff} [confidence: MEDIUM]
- Avoid: {library Z} — {why} [confidence: HIGH]

### Best Practices
- {finding with source attribution}

### Existing Art
- {reference implementation/project} — {what we can learn from it}

### Pitfalls to Avoid
- {common mistake} — {why it matters, how to prevent}

## Contradictions & Open Questions
- {topic}: Source A says X, Source B says Y. Assessment: {your reasoned take or 'needs user input'}

## Codebase Context
{only include if codebase research was done}
- Architecture: {summary of structure, boundaries, entry points}
- Key patterns: {conventions, frameworks, abstractions in use}
- Dependencies: {notable version constraints, external integrations}
- Test coverage: {state of testing infrastructure}

## Implications for Design
- {design decision this research informs}
- {constraint this research reveals}
- {opportunity this research surfaces}

## Sources
- [Title](URL) — {one-line what it contributed}"
)
```

### Write Research Brief

Write the synthesizer's output to:

```
context/refs/research-brief-{topic}.md
```

## Step 6: Present Summary

Read the research brief and present a condensed summary:

```
Research complete ({N} agents, {codebase|web|both}). Key findings:
- {3-5 most important bullet points from Summary and Key Findings}

Open questions: {list from Contradictions & Open Questions, or "none"}

Full brief: context/refs/research-brief-{topic}.md
Raw findings: context/refs/research-{topic}/
```

---

## Error Handling

- If a web agent fails to find relevant results, note the gap — do not fabricate findings
- If the codebase is empty and `--codebase-only` was set, warn: "No source files found. Nothing to research."
- If ALL agents return empty results, tell the user: "Research didn't surface actionable findings for this topic. Proceeding without a research brief."
- If the synthesizer output is malformed, write what you have and note the issue in the summary
