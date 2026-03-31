---
created: "2026-03-31T00:00:00Z"
last_edited: "2026-03-31T00:00:00Z"
---
# Implementation Tracking: Speculative Pre-Build Review

| Task | Status | Notes |
|------|--------|-------|
| T-201 | DONE | Config schema: speculative_review (on/off), speculative_review_timeout (300s). codex-speculative.sh |
| T-202 | DONE | Session-scoped job tracking via temp files. bp_speculative_record_job, bp_speculative_get_job |
| T-203 | DONE | bp_speculative_dispatch launches codex-review.sh in background at tier completion |
| T-204 | DONE | bp_speculative_status reports pipeline state (running/complete/failed with elapsed time) |
| T-205 | DONE | bp_speculative_retrieve with configurable timeout, fallback to synchronous on timeout/failure |
| T-206 | DONE | bp_speculative_reconcile merges speculative findings into tier gate flow |
| T-207 | DONE | bp_speculative_queue_finding / bp_speculative_drain_queue for P0/P1 during active tier |
| T-208 | DONE | bp_speculative_log_tier tracks review source and time-saved per tier, bp_speculative_time_saved calculates overlap |
