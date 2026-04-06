---
name: ck-design
description: "Create or update the project's DESIGN.md — the visual design system spec that constrains all UI implementation"
argument-hint: "[--import <name>] [--from-site <url>] [--audit] [--section <1-9>]"
---

> **Note:** `/bp:design` is deprecated and will be removed in a future version. Use `/ck:design` instead.

# Cavekit Design — Create or Update DESIGN.md

Create, import, update, or audit the project's visual design system document. DESIGN.md follows the 9-section Google Stitch format and serves as the authoritative visual reference for all UI-building agents across the DABI pipeline.

**DESIGN.md is a parallel constraint layer** — it is not a cavekit or a plan, but a cross-cutting visual specification that every phase consults. Think of it as CLAUDE.md for visual design.

## Step 0: Resolve Execution Profile

Before doing any substantive work:

1. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" summary` and print that exact line once.
2. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" model exploration` and store it as `EXPLORATION_MODEL`.
3. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" model reasoning` and store it as `REASONING_MODEL`.

Keep the user Q&A in the parent thread. Use `EXPLORATION_MODEL` for research and `REASONING_MODEL` for design system generation and review.

## Step 1: Check Existing State

1. Check for existing `DESIGN.md` at project root
2. Check for `context/designs/DESIGN.md`
3. If one exists, read it and present a brief summary:
   ```
   Found existing DESIGN.md ({line count} lines, {section count}/9 sections).
   Summary: {atmosphere description from Section 1}
   ```
4. Create `context/designs/` directory if missing

## Step 2: Parse Arguments

Extract from `$ARGUMENTS`:
- `--import <name>` → **Import mode** (Step 3a): fetch a design system from the awesome-design-md collection
- `--from-site <url>` → **Extract mode** (Step 3b): analyze a live site and generate DESIGN.md
- `--audit` → **Audit mode** (Step 5 only): run quality check without modifications
- `--section <1-9>` → **Update mode**: update a specific section of an existing DESIGN.md
- No arguments → **Interactive mode** (Step 3c): collaborative design conversation

If `--audit` is set and no DESIGN.md exists, report: "No DESIGN.md found. Run `/ck:design` to create one." and stop.

If `--section` is set and no DESIGN.md exists, report: "No DESIGN.md found. Run `/ck:design` to create the full design system first." and stop.

## Step 3a: Import from Collection

Fetch a curated design system from the [awesome-design-md](https://github.com/VoltAgent/awesome-design-md) repository.

1. **Resolve the name:** The `<name>` argument is a short identifier (e.g., `claude`, `vercel`, `stripe`, `github`, `linear`, `notion`, `figma`).

2. **Fetch the template:** Use WebFetch to download:
   ```
   https://raw.githubusercontent.com/VoltAgent/awesome-design-md/main/design-md/{name}/DESIGN.md
   ```
   If the fetch fails, try common path variations (`{name}/DESIGN.md`, `{Name}/DESIGN.md`). If still not found, report what names are available by fetching the repo's README.

3. **Present as starting point:**
   ```
   Imported the {Name} design system ({line count} lines).
   This is a starting point — let's customize it for your project.
   ```

4. **Walk through customization:** For each of the 9 sections, present the imported content and ask:
   - "Keep as-is, modify, or replace?"
   - For sections the user wants to modify, ask what should change
   - **Prefer multiple choice** when possible (e.g., "The imported palette uses blue as primary. Your project could: (a) keep blue, (b) switch to your brand color, (c) start fresh")

5. After customization, proceed to Step 4 (Generate).

## Step 3b: Extract from Existing Site

Analyze a live website to reverse-engineer its design system.

1. **Navigate to the URL** using Playwright browser tools:
   - Take a full-page screenshot
   - Take additional screenshots of key pages if obvious (about, pricing, docs)

2. **Analyze the visual patterns:** From the screenshots, extract:
   - **Colors:** Dominant background, text colors, accent/CTA colors, semantic colors
   - **Typography:** Font families (inspect if possible), heading sizes, body text size
   - **Spacing:** Padding patterns, margins between sections
   - **Components:** Button styles, card patterns, input styles, navigation
   - **Elevation:** Shadow usage, layering patterns
   - **Responsive:** Check mobile viewport if possible

3. **Generate a draft DESIGN.md** from the analysis — this will be imperfect, so present it with disclaimers:
   ```
   Extracted design patterns from {url}. This is a best-effort analysis —
   font families and exact values may need adjustment.
   
   Let me walk through each section for your review.
   ```

4. **Present section-by-section** for validation and correction. Proceed to Step 4 (Generate).

## Step 3c: Interactive Design Conversation

Build the design system collaboratively from scratch through conversation.

### 3c-i: Offer Visual Companion

If the visual companion can enhance the conversation (showing color swatches, typography samples, component previews), offer it:

> "I can show you color palettes, typography samples, and component previews in a browser window as we design. Want to try it? (Requires opening a local URL)"

Wait for response. If accepted, read `references/visual-companion.md` for the detailed guide. The server lives in `scripts/visual-companion/`.

### 3c-ii: Design Foundation Questions

Ask **one question at a time** to establish the design direction. Start with:

1. **Brand/Project identity:** "What is this project? Who is it for? What feeling should it evoke?"
2. **Existing assets:** "Do you have existing brand colors, logos, or fonts to work with?"
3. **Reference points:** "Are there any existing products whose visual style you admire or want to be similar to?"
4. **Constraints:** "Any technical constraints? (e.g., must work in dark mode, must use system fonts, must support RTL)"

### 3c-iii: Build Sections Incrementally

Walk through each section, scaling depth to complexity:

**Section 1 — Visual Theme & Atmosphere:**
- Propose 2-3 atmosphere directions based on the foundation answers
- Use the visual companion to show mood boards if accepted
- Get approval before moving on

**Section 2 — Color Palette:**
- Propose a palette based on brand inputs or reference points
- Show swatches via visual companion if available
- Include primary, neutral, and semantic colors
- Ask about dark mode support

**Section 3 — Typography:**
- Propose font pairings (heading + body + code)
- Show type scale samples if visual companion is available
- Define the complete scale table

**Sections 4-6 — Components, Layout, Elevation:**
- Build on the established colors, fonts, and spacing
- Show component previews via visual companion
- One section at a time, approval before advancing

**Section 7 — Do's and Don'ts:**
- Synthesize from decisions made in previous sections
- Propose concrete code examples

**Section 8 — Responsive:**
- Define breakpoints based on the project's target devices
- Specify mobile adaptations for key components

**Section 9 — Agent Prompt Guide:**
- Auto-generate from the previous 8 sections
- Quick reference with most-used values
- Example prompts demonstrating proper token usage

## Step 4: Generate DESIGN.md

**Only after user approves the design**, write the file.

Dispatch a subagent with `model: "{REASONING_MODEL}"` to write the complete DESIGN.md:

```
Agent(
  subagent_type: "general-purpose",
  model: "{REASONING_MODEL}",
  description: "Write DESIGN.md",
  prompt: |
    Write a complete DESIGN.md file following the 9-section Google Stitch format.
    
    Use the ck:design-system skill for the format template and quality standards.
    
    DESIGN DECISIONS:
    {all approved design decisions from the conversation}
    
    RULES:
    - All 9 sections required
    - Every color needs: semantic name + hex value + functional role
    - Complete type scale table with all 5 values per level
    - Component stylings include hover, focus, active, disabled states
    - Spacing scale uses a consistent base unit
    - Do's and Don'ts have concrete code examples
    - Agent Prompt Guide has quick reference + example prompts
    - Add YAML frontmatter: created and last_edited dates (ISO 8601 UTC)
    
    Write the file to {project root}/DESIGN.md.
)
```

## Step 5: Design Review Loop

After writing, dispatch a **design-reviewer** subagent to verify quality:

```
Agent(
  subagent_type: "ck:design-reviewer",
  model: "{REASONING_MODEL}",
  description: "Review DESIGN.md",
  prompt: |
    Review the DESIGN.md at the project root for completeness, consistency,
    and actionability. Follow the review criteria in your agent definition.
)
```

**Review loop rules:**
- If **Approved**: proceed to Step 6
- If **Issues Found**: fix the issues, re-dispatch the reviewer
- Maximum **3 iterations** — if still not passing, present remaining issues to the user for guidance

## Step 6: User Review Gate

Present the DESIGN.md for final approval:

> "Design system written and validated. The file is at `DESIGN.md` in your project root. Please review and let me know if you want any changes."

Wait for response. If changes requested, make them and re-run Step 5.

## Step 7: Finalize and Report

1. **Ensure context directory:** Create `context/designs/` if missing
2. **Write context CLAUDE.md** (if missing):
   ```markdown
   # Design System

   The project's visual design system in DESIGN.md format (9-section Google Stitch).

   ## Conventions
   - DESIGN.md at project root is the canonical source
   - All UI implementation must reference DESIGN.md tokens and patterns
   - Updated via /ck:design or automatically during /ck:check and /ck:revise
   - Agents read this before implementing any user-facing component
   ```

3. **Initialize changelog** (if missing): Create `context/designs/design-changelog.md`:
   ```markdown
   # Design System Changelog

   Append-only log of DESIGN.md changes.

   | Date | Section | Change | Source |
   |------|---------|--------|--------|
   | {today} | All | Initial design system created | /ck:design |
   ```

4. **Report:**
   ```markdown
   ## Design System Report

   **File:** DESIGN.md ({line count} lines)
   **Sections:** 9/9
   **Colors:** {count} defined
   **Type Scale Levels:** {count}
   **Components:** {list}
   **Responsive Breakpoints:** {count}

   ### Next Steps
   - Run `/ck:sketch` to create kits — they'll reference this design system
   - Run `/ck:design --audit` anytime to check design system health
   - Run `/ck:design --section N` to update a specific section
   ```

---

## Update Mode (`--section <1-9>`)

When updating a specific section of an existing DESIGN.md:

1. Read the current DESIGN.md
2. Present the current content of the specified section
3. Ask what should change
4. Update only that section (preserve all other sections)
5. Run the design review loop (Step 5) to verify consistency
6. Log the change to `context/designs/design-changelog.md`

---

## Key Principles

- **One question at a time** — do not overwhelm with multiple design decisions
- **Multiple choice preferred** — show 2-3 options for colors, fonts, layouts
- **Visual companion when possible** — design is visual, text is a lossy medium
- **Concrete values only** — every decision results in a specific hex, px, or token value
- **Build incrementally** — section by section, approval before advancing
- **Living document** — DESIGN.md evolves through inspect and revise, not just this command
