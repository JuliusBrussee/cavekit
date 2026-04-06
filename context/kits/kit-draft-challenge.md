---
created: "2026-03-31T00:00:00Z"
last_edited: "2026-03-31T00:00:00Z"
---

# Cavekit: Dual-Model Cavekit Drafting (Design Challenge)

## Scope
After Claude drafts kits and the cavekit-reviewer passes them, send the kits to Codex for a "Design Challenge" review before presenting them to the user. Codex examines the domain decomposition, requirement coverage, and specification quality from a fundamentally different model perspective — catching blind spots Claude's own reviewer cannot.

## Requirements

### R1: Design Challenge Invocation
Send completed kits to Codex for adversarial design review.
- [ ] After the cavekit-reviewer agent approves (Step 8 of the draft process), invoke Codex with all cavekit files and the overview
- [ ] The Codex prompt specifically targets design-level concerns: domain decomposition quality, missing domains, over/under-decomposition, ambiguous requirements, untestable acceptance criteria, scope gaps, and implicit assumptions
- [ ] Codex reviews kits as a whole system, not individual files — cross-domain coherence is the primary concern
- [ ] The design challenge runs synchronously (the user should not see kits that haven't been challenged)

### R2: Challenge Output Format
Codex produces structured design feedback.
- [ ] Findings categorized as: `decomposition` (wrong domain boundaries), `coverage` (missing requirements), `ambiguity` (unclear acceptance criteria), `scope` (over/under-scoped), `assumption` (implicit assumptions that should be explicit)
- [ ] Each finding references specific cavekit files and requirement numbers
- [ ] Findings rated as `critical` (must fix before architect phase) or `advisory` (worth considering)
- [ ] If Codex finds no critical issues, it returns an explicit approval

### R3: Auto-Fix and Re-Challenge Loop
Claude addresses critical findings automatically.
- [ ] Critical findings are addressed by Claude (edit kits, add missing requirements, clarify acceptance criteria)
- [ ] After fixes, re-send to Codex for a second challenge pass
- [ ] Maximum 2 challenge-fix cycles — after that, present remaining critical findings to the user for judgment
- [ ] Advisory findings are collected and presented to the user alongside the kits (not auto-fixed)

### R4: Draft Flow Integration
The design challenge integrates into the existing `/ck:sketch` command flow.
- [ ] Inserted between Step 8 (cavekit-reviewer loop) and Step 9 (user review gate)
- [ ] When Codex is unavailable, skip the design challenge — the cavekit-reviewer loop alone is sufficient (graceful degradation)
- [ ] The user review gate (Step 9) now shows both the kits AND any advisory findings from Codex, so the user has full context
- [ ] Time added to the draft phase is logged (Codex challenge duration)

### R5: Challenge Prompt Design
The prompt sent to Codex is purpose-built for design-level review.
- [ ] The prompt includes: all cavekit files, the overview, and explicit instructions to challenge (not rubber-stamp)
- [ ] Codex is instructed to propose at least one alternative decomposition if it can identify a better one
- [ ] Codex is instructed to look for requirements that are technically testable but practically meaningless ("checkbox requirements")
- [ ] The prompt explicitly prohibits implementation-level feedback (no framework suggestions, no file path opinions)

## Out of Scope
- Codex writing kits from scratch (Claude drafts, Codex challenges)
- Codex reviewing implementation plans or build sites (that's the tier gate / inspector's job)
- Multi-model consensus (only Claude + Codex, not additional models)
- Challenging individual cavekit revisions (only the initial draft and `/ck:revise` outputs)

## Cross-References
- See also: cavekit-codex-bridge.md (R1 for detection, R3 for invocation mechanism)
- See also: agents/cavekit-reviewer.md (runs before the design challenge)
- See also: skills/peer-review/SKILL.md (Design Challenge is one of the six review modes)

## Changelog
- 2026-03-31: Initial draft
