---
created: "2026-03-31T00:00:00Z"
last_edited: "2026-03-31T00:00:00Z"
---
# Implementation Tracking: Codex Bridge + Tier Gate

| Task | Status | Notes |
|------|--------|-------|
| T-001 | DONE | Codex binary detection via `codex --version`. scripts/codex-detect.sh |
| T-002 | DONE | Plugin presence check across standard Claude Code plugin locations. scripts/codex-detect.sh |
| T-003 | DONE | Config schema: codex_review (auto/off), codex_model, codex_effort. scripts/codex-config.sh |
| T-004 | DONE | Gate config: tier_gate_mode (severity/strict/permissive/off). scripts/codex-config.sh |
| T-005 | DONE | codex_available flag and bp_codex_nudge already in codex-detect.sh |
| T-006 | DONE | Codex review invocation with finding parser. scripts/codex-review.sh |
| T-007 | DONE | Finding format with source/tier columns. scripts/codex-findings.sh |
| T-008 | DONE | Peer-review-loop replacement: Codex CLI primary, MCP legacy fallback. setup-build.sh + SKILL.md |
| T-009 | DONE | /ck:judge standalone command. commands/codex-review.md |
| T-010 | DONE | Tier boundary hook in build loop. commands/build.md |
| T-011 | DONE | Severity-based gating with fix-task generation. scripts/codex-gate.sh + build.md |
| T-012 | DONE | Review-fix cycle with re-review and max iterations. bp_review_fix_cycle in codex-gate.sh |
