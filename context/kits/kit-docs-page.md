---
created: "2026-04-01T11:50:00Z"
last_edited: "2026-04-01T11:50:00Z"
---

# Cavekit: Documentation Page

## Scope

A single-file (`docs.html`) documentation page with sticky sidebar navigation, deployed alongside `index.html` on the `gh-pages` branch. Contains the full Cavekit reference: overview, quick start, commands, methodology, Codex integration, skills reference, configuration, and file structure. Shares the same visual identity as the landing page. Zero dependencies, zero build step.

## Requirements

### R1: Shared Visual Identity
**Description:** The docs page must use the same design language as the landing page for a cohesive experience.
**Acceptance Criteria:**
- [ ] Same CSS custom properties (color palette, fonts, glassmorphism values) as landing page R1
- [ ] Same background treatment (near-black with noise texture + faint grid)
- [ ] Same font stack (display font for headings, monospace for code/annotations, system sans for body)
- [ ] Glassmorphism card and surface treatments match landing page
- [ ] `-webkit-font-smoothing: antialiased` applied to root

### R2: Page Layout
**Description:** Three-column docs layout with sticky navigation.
**Acceptance Criteria:**
- [ ] Desktop (>=1024px): left sidebar (240px, sticky) + main content (flexible, max ~780px) + right sidebar (200px, sticky "On this page" TOC)
- [ ] Tablet (<1024px): left sidebar collapses to hamburger menu overlay, right sidebar hidden, main content full-width
- [ ] Mobile (<768px): full-width content, hamburger nav menu
- [ ] Main content area has comfortable reading line-length (60-75 characters)
- [ ] Content area centered with consistent horizontal padding

### R3: Top Bar
**Description:** Fixed top navigation bar.
**Acceptance Criteria:**
- [ ] Fixed at top of viewport, glassmorphism blur background
- [ ] Left: "Cavekit" text logo (links to `index.html`)
- [ ] Right: "GitHub" link + version badge pill `v2.1.0`
- [ ] Hamburger menu button visible on tablet/mobile (hidden on desktop)
- [ ] Height ~56px, z-index above sidebar and content

### R4: Left Sidebar Navigation
**Description:** Sticky sidebar with the full documentation tree and scrollspy.
**Acceptance Criteria:**
- [ ] Sticky positioning, scrolls independently of main content
- [ ] Top: filter input (monospace, subtle border, placeholder "Filter...") that narrows visible nav items via substring match
- [ ] Navigation tree with collapsible sections (chevron indicators):
  - Overview
  - Quick Start (children: Greenfield, Brownfield)
  - Commands (children: all 10 `/ck:*` commands)
  - Methodology (children: DABI Lifecycle, Kits as Source of Truth, Scientific Method Applied)
  - Codex Integration (children: Design Challenge, Tier Gate, Speculative Review, Command Safety Gate, Graceful Degradation)
  - Skills Reference (children: all 13 skills)
  - Configuration (children: Settings Reference, File Structure)
- [ ] Bottom: "Back to Home" link to `index.html`
- [ ] Scrollspy: active section highlighted with blue accent (left border or text color change) as user scrolls through content
- [ ] Clicking a nav item smooth-scrolls to that section
- [ ] URL hash updates on scroll and on nav click for deep linking (e.g., `docs.html#commands-bp-draft`)

### R5: Right Sidebar (On This Page)
**Description:** Contextual table of contents for the currently active section.
**Acceptance Criteria:**
- [ ] Shows H3 headings within the current active top-level section
- [ ] Scrollspy highlights the current H3
- [ ] Clicking scrolls to that H3
- [ ] Updates when the user scrolls into a different top-level section
- [ ] Hidden on tablet and mobile (right sidebar only visible >=1024px)
- [ ] Heading: "On this page" in monospace, muted

### R6: Overview Section
**Description:** Introduction to what Cavekit is and the DABI lifecycle.
**Acceptance Criteria:**
- [ ] Content explains: what Cavekit is, who it's for, the core idea (specification layer between intent and code)
- [ ] Brief DABI lifecycle summary with the 4 phases
- [ ] Links to each phase's detailed section in Commands and Methodology
- [ ] Content derived from README "The Idea" and "How It Works" intro sections

### R7: Quick Start Section
**Description:** Getting started guides for greenfield and brownfield projects.
**Acceptance Criteria:**
- [ ] Two subsections: Greenfield and Brownfield
- [ ] Each shows a full annotated conversation example (terminal-style code blocks)
- [ ] Greenfield example: `/ck:sketch` → `/ck:map` → `/ck:make` with sample output
- [ ] Brownfield example: `/ck:sketch --from-code` → `/ck:map --filter` → `/ck:make` with sample output
- [ ] Content derived from README "Quick Start" section

### R8: Commands Section
**Description:** Reference for all Cavekit slash commands.
**Acceptance Criteria:**
- [ ] Each command gets its own subsection with:
  - Command name in monospace heading (e.g., `/ck:sketch`)
  - Phase badge (Draft, Architect, Build, Inspect, or "Utility")
  - One-line description
  - Usage example in terminal code block
  - Flags/options table if applicable (e.g., `--from-code`, `--filter`)
  - Links to related commands
- [ ] Commands covered: `/ck:sketch`, `/ck:map`, `/ck:make`, `/ck:check`, `/ck:research`, `/ck:progress`, `/ck:scan`, `/ck:revise`, `/ck:judge`, `/ck:help`
- [ ] Content derived from README "Commands" section + individual command descriptions

### R9: Methodology Section
**Description:** The Cavekit methodology and philosophy.
**Acceptance Criteria:**
- [ ] Three subsections: The DABI Lifecycle, Kits as Source of Truth, Scientific Method Applied
- [ ] DABI Lifecycle: detailed walkthrough of each phase with what it produces and why
- [ ] Kits as Source of Truth: explains why specs drive development, not memory
- [ ] Scientific Method: maps hypothesis→test→observe→refine to kits→gates→loops→revision
- [ ] Content derived from README "Methodology" and "Why Cavekit" sections

### R10: Codex Integration Section
**Description:** Full documentation of the Codex adversarial review system.
**Acceptance Criteria:**
- [ ] Five subsections: Design Challenge, Tier Gate, Speculative Review, Command Safety Gate, Graceful Degradation
- [ ] Each subsection explains: what it does, when it triggers, how it works, configuration options
- [ ] Design Challenge: the pre-build cavekit review flow
- [ ] Tier Gate: severity levels (P0-P3), gate modes, fix cycle behavior
- [ ] Speculative Review: background review overlapping with build, timeout behavior
- [ ] Command Safety Gate: allowlist/blocklist, Codex classification, verdict cache
- [ ] Graceful Degradation: behavior when Codex is not installed
- [ ] Content derived from README "Codex Adversarial Review" section

### R11: Skills Reference Section
**Description:** Reference for all 13 Cavekit skills.
**Acceptance Criteria:**
- [ ] Each skill gets a card/entry with: name, one-line description, when to use it
- [ ] Skills covered: Cavekit Writing, Convergence Monitoring, Peer Review, Validation-First Design, Context Architecture, Revision, Brownfield Adoption, Speculative Pipeline, Prompt Pipeline, Implementation Tracking, Documentation Inversion, Peer Review Loop, Core Methodology
- [ ] Content derived from README skills list + individual skill descriptions in `skills/` directory

### R12: Configuration Section
**Description:** Settings reference and file structure documentation.
**Acceptance Criteria:**
- [ ] Settings Reference: table of all Codex settings (codex_review, codex_model, tier_gate_mode, command_gate, command_gate_timeout, speculative_review, speculative_review_timeout) with values, defaults, and purpose
- [ ] File Structure: the `context/` directory tree with descriptions of each directory and file type
- [ ] Content derived from README "Configuration" and "File Structure" sections

### R13: Content Styling
**Description:** Consistent styling for all documentation content elements.
**Acceptance Criteria:**
- [ ] Headings (H2, H3): display font with a 3px left border-accent in blue
- [ ] Code blocks: glassmorphism terminal treatment matching landing page, with span-class-based syntax coloring (blue for commands, green for output, red for errors)
- [ ] Tables: glassmorphism rows with alternating subtle background opacity
- [ ] Inline code: monospace with subtle blue background pill
- [ ] Links: accent blue, underline on hover
- [ ] Command entries: card format with glassmorphism surface
- [ ] `text-wrap: balance` on headings, `text-wrap: pretty` on body paragraphs
- [ ] `font-variant-numeric: tabular-nums` on any numeric content

### R14: Responsive Design
**Description:** The docs page must work across all viewport sizes.
**Acceptance Criteria:**
- [ ] Breakpoints: mobile (<768px), tablet (768-1023px), desktop (>=1024px)
- [ ] Layout adapts per R2 (three-column → two-column → single-column)
- [ ] Hamburger menu on tablet/mobile opens a slide-in overlay with the full nav tree
- [ ] Code blocks use `overflow-x: auto` for horizontal scroll on narrow viewports
- [ ] Tables scroll horizontally in a container on mobile
- [ ] Body text minimum 16px
- [ ] `viewport` meta tag: `width=device-width, initial-scale=1`
- [ ] No horizontal scrollbar on page body at any viewport

### R15: Accessibility
**Description:** The docs page must be accessible.
**Acceptance Criteria:**
- [ ] Semantic HTML: `<nav>` for sidebars, `<main>` for content, `<section>` for each docs section
- [ ] All text meets WCAG AA contrast ratio (4.5:1) against background
- [ ] Keyboard-navigable: all nav items, links, buttons, and the filter input reachable via Tab
- [ ] Visible focus indicators on all interactive elements
- [ ] `aria-current="true"` on the active scrollspy nav item
- [ ] Hamburger menu is a `<button>` with `aria-expanded` and `aria-controls`
- [ ] Filter input has an associated `<label>` (can be visually hidden)
- [ ] Skip-to-content link for keyboard users
- [ ] Heading hierarchy: h1 (page title) → h2 (sections) → h3 (subsections), no skips

### R16: Performance
**Description:** The docs page must load fast.
**Acceptance Criteria:**
- [ ] Single `docs.html` file — all CSS inline in `<style>`, all JS inline in `<script>`
- [ ] Total file size under 80KB (uncompressed) — more content than landing page, but still lean
- [ ] No external dependencies except optional Google Fonts (shared with landing page, `font-display: swap`)
- [ ] JS is minimal: IntersectionObserver for scrollspy, sidebar toggle, filter function, hash management
- [ ] No images — all visuals are CSS or inline SVG

## Out of Scope
- No full-text search (filter is substring match on nav item names only)
- No JavaScript framework or build system
- No versioned docs (single version only)
- No edit-on-GitHub links
- No analytics or tracking
- Content is a reformatted presentation of existing README and skill files — no new technical writing

## Cross-References
- See also: cavekit-landing-page.md (shares visual identity system, linked from docs top bar and "Back to Home")
- Landing page R1 (Visual Identity) is the source of truth for shared design tokens

## Changelog
