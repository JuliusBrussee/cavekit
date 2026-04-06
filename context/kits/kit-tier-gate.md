---
created: "2026-03-31T00:00:00Z"
last_edited: "2026-03-31T00:00:00Z"
---

# Cavekit: Tier Gate

## Scope
The mechanism that invokes Codex adversarial review at the end of every build tier and decides whether to block or proceed based on finding severity.

## Requirements

### R1: Tier Boundary Hook
Invoke Codex adversarial review automatically at the end of every completed tier during `/ck:make`.
- [ ] After all tasks in a tier are merged and impl tracking is updated, trigger a Codex review of the tier's cumulative diff
- [ ] The diff target is: worktree base to current HEAD (captures everything built in this tier)
- [ ] The review runs inline (not background) — the build loop waits for results before advancing
- [ ] Skip the review if Codex is unavailable (graceful degradation — log a note and continue)

### R2: Severity-Based Gating
Block or proceed based on finding severity, with configurable override.
- [ ] P0/P1 findings block the tier — the build loop must address them before advancing to the next tier
- [ ] P2/P3 findings are logged to `impl-review-findings.md` and the build proceeds
- [ ] When blocked, the build loop creates fix tasks for each P0/P1 finding and executes them as an additional wave within the current tier
- [ ] After fix tasks complete, re-run the Codex review on the updated diff
- [ ] Maximum 2 review-fix cycles per tier — after that, log remaining P0/P1 findings and advance with a warning

### R3: Gate Configuration
User settings to control gating behavior.
- [ ] Setting `tier_gate_mode` with values `"severity" | "strict" | "permissive" | "off"`
  - `severity` (default): P0/P1 block, P2/P3 deferred
  - `strict`: all findings block
  - `permissive`: all findings deferred (log only)
  - `off`: no tier gate review
- [ ] Settings stored alongside Codex Bridge config (same config mechanism)

### R4: Finding Integration
Codex review findings are captured in Cavekit's standard tracking.
- [ ] All findings appended to `context/impl/impl-review-findings.md` with tier number and source
- [ ] Each finding includes: severity (P0-P3), file, line, description, tier number, source (`codex-tier-gate`)
- [ ] Fix tasks generated from P0/P1 findings reference the original finding ID
- [ ] After fix tasks complete, the finding is updated with resolution status

## Out of Scope
- The Codex communication mechanism itself (that's Codex Bridge's job)
- Standalone on-demand review (that's Codex Bridge R4)
- Changing the build loop's wave/frontier logic beyond adding the tier boundary hook

## Cross-References
- See also: cavekit-codex-bridge.md (R1 for detection, R3 for review invocation and finding format)
- See also: cavekit-speculative-review.md (speculative review feeds findings into tier gate gating logic)

## Changelog
- 2026-03-31: Initial draft
