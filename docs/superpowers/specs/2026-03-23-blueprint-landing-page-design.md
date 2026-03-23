# Blueprint Landing Page — Design Spec

## Overview

A static, single-page landing site for the Blueprint Claude Code plugin, deployed to GitHub Pages. The page uses an animated engineering-blueprint aesthetic — deep navy background, fine grid lines, monospace technical annotations, and SVG path animations that draw themselves as the user scrolls. The goal is to communicate what Blueprint does, why it matters, and how to install it.

**Target audience:** Developers and AI enthusiasts evaluating Claude Code plugins.

**Tech stack:** Pure HTML/CSS/JS in a single `index.html`. Zero dependencies, zero build step. SVG animations driven by CSS `@keyframes` + `stroke-dashoffset`. Scroll triggers via `IntersectionObserver`. Deployed by pushing to the `gh-pages` branch.

**Repo location:** Deployed from a dedicated `gh-pages` branch with `index.html` at branch root. The `main` branch is not affected — GitHub Pages is configured to serve from the `gh-pages` branch.

---

## Visual Identity

### Color Palette

| Token | Value | Usage |
|-------|-------|-------|
| `--bg` | `#0f1a2e` | Page background |
| `--grid-fine` | `rgba(74,158,255,0.08)` | Fine grid lines (16px) |
| `--grid-major` | `rgba(74,158,255,0.12)` | Major grid lines (80px) |
| `--blueprint` | `#4a9eff` | Primary accent — lines, borders, text |
| `--blueprint-dim` | `rgba(74,158,255,0.7)` | Annotation text, section labels |
| `--blueprint-decoration` | `rgba(74,158,255,0.3)` | Non-text decorations: divider lines, dimension lines |
| `--blueprint-fill` | `rgba(74,158,255,0.06)` | Box fills, subtle backgrounds |
| `--text-primary` | `#e8edf3` | Headings, primary copy |
| `--text-secondary` | `rgba(200,215,235,0.65)` | Body text, descriptions |
| `--red-diagnostic` | `#ff6464` | Problem section card text and bar fills |
| `--green-pass` | `#00ff88` | Success/commit indicators |

### Typography

- **Headings:** System sans-serif (`system-ui, -apple-system, sans-serif`), weight 700, tight letter-spacing (-0.5px)
- **Body:** System sans-serif, weight 400
- **Annotations/Labels:** Monospace (`'SF Mono', 'Fira Code', 'Cascadia Code', monospace`), uppercase, letter-spacing 2-3px, small size (9-10px)
- **Code/Terminal:** Monospace, regular weight

### Global Background

The entire page has a persistent blueprint grid composed of two layers:
1. Fine grid: 16px spacing, `--grid-fine` color
2. Major grid: 80px spacing, `--grid-major` color

Both are CSS `background-image` linear gradients on the `<body>`.

### Global Decorations

- Section dividers are subtle horizontal lines in `--blueprint-decoration`
- Each section has a top-left annotation in monospace `--blueprint-dim`: `SECTION 01 — HERO`, `SECTION 02 — THE PROBLEM`, `SECTION 03 — HOW IT WORKS`, `SECTION 04 — THE RALPH LOOP`, `SECTION 05 — GET STARTED`
- Dimension lines (thin lines with perpendicular end-caps) appear as SVG decorations in `--blueprint-decoration` in margins where space allows

---

## Sections

### Section 1: Hero (Viewport Height)

**Layout:** Centered vertically and horizontally, full viewport height.

**Content:**
- Title: "Blueprint" in large heading type
- Subtitle: "Specification-driven development for AI coding agents"
- The DABI pipeline schematic (primary SVG diagram — see below)
- CTA: monospace install command in a bordered box (`git clone https://github.com/JuliusBrussee/blueprint.git ~/.blueprint && cd ~/.blueprint && ./install.sh`)
- Secondary link: "View on GitHub"

**DABI Pipeline Schematic (Hero SVG):**

The centerpiece. An SVG diagram showing the full flow:

```
[YOU] ---> [DRAFT] ---> [ARCHITECT] --+--> [AGENT 1] --+
           Blueprints    Build Site   +--> [AGENT 2] --+--> [MERGE] --> main
                                      +--> [AGENT 3] --+
```

- Boxes are drawn with `stroke-dashoffset` animation (border draws itself)
- Connecting lines animate as dashed paths extending left-to-right
- Fan-out lines to agents draw simultaneously
- Merge lines converge
- Dimension line underneath: "AUTOMATED PIPELINE"
- Small annotation labels above boxes: "BLUEPRINTS", "BUILD SITE", "MAIN"

**Animation (on page load):**
1. Grid fades in (200ms)
2. Title and subtitle fade in (400ms, slight upward translate)
3. SVG pipeline draws itself left-to-right over ~2.5s using staggered `animation-delay`
4. CTA fades in after pipeline completes

### Section 2: The Problem

**Layout:** 2x2 grid of diagnostic cards.

**Content (4 pain points):**
1. "Context Lost" — agents forget what they said 3 steps ago
2. "No Validation" — code gets written but never verified against intent
3. "Single Agent" — one agent, one task, one branch
4. "No Iteration" — single pass produces rough drafts, not production code

Each card has:
- A dashed red border
- The pain point name in monospace red
- A progress/severity bar that fills to a specific width
- A one-line description beneath

**Animation (on scroll into view):**
- Cards fade in with stagger (100ms between each)
- Red diagnostic bars fill from 0% to their target width over 800ms
- Bars use `ease-out` timing for a satisfying deceleration

### Section 3: How It Works — DABI Phases

**Layout:** 4 equal-width cards in a horizontal row with arrow separators.

**Content (one card per phase):**
1. **D** — Draft: "Define the what" + `/bp:draft`
2. **A** — Architect: "Plan the order" + `/bp:architect`
3. **B** — Build: "Run the loop" + `/bp:build`
4. **I** — Inspect: "Verify the result" + `/bp:inspect`

Each card has a large letter, the phase name, and a one-line description. Connecting arrows (`→`) sit between cards.

Below the 4 cards: a brief paragraph explaining the flow — blueprints are the source of truth, agents build from them, validation traces back to requirements.

**Animation (on scroll into view):**
- Each card's border draws itself (stroke-dashoffset) from left to right
- Content fades in after its card border completes
- Staggered: 300ms delay between each card
- Arrows fade in between cards as the corresponding card completes

### Section 4: The Ralph Loop

**Layout:** Centered circular/elliptical SVG diagram with explanatory text below.

**Content:**

An elliptical loop with 5 nodes positioned around it:
1. READ (read spec + find next task)
2. IMPLEMENT (write code)
3. VALIDATE (build + test + acceptance criteria)
4. COMMIT (atomic commit if pass)
5. NEXT TASK (advance to next unblocked task)

A "FAIL → fix → retry" branch off the VALIDATE node.

Below the diagram: "Each iteration validates against the blueprint. The loop runs until all tasks pass or the iteration limit is reached."

Below the explanatory text: a counter animation in monospace `--blueprint-dim`, centered, showing "Iteration 1... 2... 3... 18 — COMPLETE". Counts from 1 to 18 at 150ms intervals, pauses 500ms on "COMPLETE", then loops.

**Animation (on scroll into view):**
- The elliptical loop path draws itself
- A glowing particle (small circle with box-shadow glow) continuously traces the loop path using SVG `<animateMotion>`
- Each node pulses (opacity/scale) as the particle passes through it
- The COMMIT node briefly flashes green when the particle passes

### Section 5: Get Started

**Layout:** Centered terminal-style block with action buttons below.

**Content:**
- A dark terminal box with two lines:
  ```
  $ git clone https://github.com/JuliusBrussee/blueprint.git ~/.blueprint
  $ cd ~/.blueprint && ./install.sh
  ```
- Below: "Requires Claude Code, git, macOS/Linux. Optional: tmux for parallel agents."
- Two buttons: "View on GitHub →" (primary) and "MIT License" (secondary/muted)

**Animation (on scroll into view):**
- Terminal prompt (`$`) appears
- First command types out character by character (~50ms per char)
- Brief pause (300ms)
- Second command types out
- Buttons fade in after typing completes

---

## Animation Architecture

### Scroll Triggering

All section animations are triggered by `IntersectionObserver` with `threshold: 0.15` (triggers when 15% of the section is visible). Each section's root element gets a `.visible` class added, which activates CSS animations on child elements.

```
.section:not(.visible) .animate-target {
  opacity: 0;
}
.section.visible .animate-target {
  animation: fadeDrawIn 0.8s ease-out forwards;
}
```

### SVG Path Drawing

All SVG line/path animations use the stroke-dasharray/dashoffset technique:

```css
.draw-path {
  stroke-dasharray: var(--path-length);
  stroke-dashoffset: var(--path-length);
}
.visible .draw-path {
  animation: drawLine 1.5s ease-out forwards;
}
@keyframes drawLine {
  to { stroke-dashoffset: 0; }
}
```

Path lengths are hardcoded per-path (calculated from SVG geometry).

### Typewriter Effect

The install section uses a JS-driven typewriter. On `.visible`, a function iterates through characters with `setTimeout`, inserting one character per tick into a `<span>`. A blinking cursor (`|`) follows the insertion point via CSS `@keyframes blink`.

### Performance

- All animations use `transform` and `opacity` only (GPU-composited, no layout thrashing)
- SVG animations use `stroke-dashoffset` (also compositable)
- The orbiting particle in the Ralph Loop uses SVG `<animateMotion>` along the elliptical path (universal browser support, no fallback needed)
- No animation libraries. Total JS: IntersectionObserver setup + typewriter function (~50 lines)
- Reduced motion: `@media (prefers-reduced-motion: reduce)` disables all animations, shows all content immediately

---

## Responsive Design

### Desktop (>768px)
- Max content width: 900px, centered
- Hero SVG: full width
- Problem cards: 2x2 grid
- DABI cards: 4-column row
- Ralph Loop: full-size elliptical diagram

### Mobile (<768px)
- Hero SVG: horizontally scrollable in an `overflow-x: auto` container with `-webkit-overflow-scrolling: touch`
- Problem cards: single column stack (1x4)
- DABI cards: vertical stack with downward arrows (`↓`) replacing horizontal arrows
- Ralph Loop: scales down proportionally via `viewBox` (no layout change — SVG handles it)
- Terminal: `font-size: 11px`, `overflow-x: auto` with horizontal scroll for long commands

---

## Deployment

- Single `index.html` on the `gh-pages` branch (branch root)
- GitHub Pages configured to serve from `gh-pages` branch
- The `main` branch is unaffected — landing page lives only on `gh-pages`
- No build step — push `index.html` to `gh-pages` and it's live
- Custom domain optional (add a `CNAME` file to `gh-pages` branch root)

### Footer

A minimal footer at the bottom of the page:
- "Blueprint — MIT License" in `--text-secondary`
- "Built by Julius Brussee" link to GitHub profile
- Styled in monospace, small size, centered

---

## Accessibility

- All SVG diagrams have `role="img"` and `aria-label` descriptions
- `@media (prefers-reduced-motion: reduce)` shows all content without animation
- Color contrast: all text tokens (`--text-primary`, `--text-secondary`, `--blueprint`, `--blueprint-dim`, `--red-diagnostic`) meet WCAG AA 4.5:1 against `--bg`. Decorative-only elements (`--blueprint-decoration`, `--grid-fine`, `--grid-major`) are exempt.
- Semantic HTML: `<header>`, `<section>`, `<footer>` with `aria-labelledby`
- Keyboard-navigable links and buttons with visible focus indicators
- Typewriter text has a `<noscript>` fallback showing the full command immediately

---

## Out of Scope

- No JavaScript framework or build system
- No analytics or tracking
- No dark/light mode toggle (blueprint aesthetic is inherently dark)
- No interactive elements beyond links (no forms, no search)
- No blog, docs, or multi-page structure
- Content comes from the existing README — no new copywriting required beyond condensing for the landing page format
