# Session Feedback Protocol Reference

Inter-session communication format for AI agents. Covers the XML feedback format, work item fields, work queue handoff, and session continuity patterns.

---

## 1. Overview

AI agent sessions are stateless -- each session starts with no memory of prior sessions. The session feedback protocol bridges this gap by providing a structured format for communicating session results and next-session work queues.

Two components work together:
1. **Session Feedback XML** -- structured report of what happened in a session
2. **Work Queue Handoff** -- prioritized list of work items for the next session

Together, they let the next session start producing immediately — no ramp-up time spent figuring out what to work on.

---

## 2. Session Feedback XML Format

### Structure

```xml
<session-report>
  <task id="T-1">
    <status>DONE</status>
    <files-modified>true</files-modified>
    <summary>Implemented authentication module with login, register, and session management</summary>
    <obstacles kind="NONE"></obstacles>
    <next-steps>Proceed to API integration in next session</next-steps>
  </task>
  <task id="T-2">
    <status>PARTIAL</status>
    <files-modified>true</files-modified>
    <summary>Dashboard data display implemented, filtering partially complete</summary>
    <obstacles kind="TEMPORARY">Waiting for API endpoint documentation to implement advanced filters</obstacles>
    <next-steps>Complete filtering once API docs are available; move to export feature</next-steps>
  </task>
  <task id="T-3">
    <status>BLOCKED</status>
    <files-modified>false</files-modified>
    <summary>Could not start performance optimization</summary>
    <obstacles kind="PERMANENT">Requires database indexing strategy that needs DBA review</obstacles>
    <next-steps>Escalate to human for DBA consultation before attempting again</next-steps>
  </task>
</session-report>
```

### Element Reference

| Element | Required | Description |
|---------|----------|-------------|
| `<session-report>` | Yes | Root element containing all work items |
| `<task id="T-N">` | Yes | Work item container with unique ID |
| `<status>` | Yes | One of: `DONE`, `PARTIAL`, `BLOCKED` |
| `<files-modified>` | Yes | `true` if code was modified, `false` otherwise |
| `<summary>` | Yes | Brief description of what was accomplished |
| `<obstacles>` | Yes | Empty if none; description if blocked. `kind` attribute required. |
| `<next-steps>` | Yes | What the next session should do with this work item |

---

## 3. Status Values

### DONE

The work item is complete. All acceptance criteria are met, tests pass, and no further work is needed on this item.

```xml
<task id="T-1">
  <status>DONE</status>
  <files-modified>true</files-modified>
  <summary>Authentication module fully implemented: login, register, session management, logout. All 12 tests passing.</summary>
  <obstacles kind="NONE"></obstacles>
  <next-steps>No further work needed. Integration tests with API module should be added when API is ready.</next-steps>
</task>
```

**Next session behavior:** Skip this work item. It is complete.

### PARTIAL

The work item was started but not finished. Some progress was made.

```xml
<task id="T-2">
  <status>PARTIAL</status>
  <files-modified>true</files-modified>
  <summary>Dashboard layout and data display working. Filtering UI created but filter logic not connected to API. 8/12 acceptance criteria met.</summary>
  <obstacles kind="TEMPORARY">API filter endpoint documentation not available during this session</obstacles>
  <next-steps>Connect filter logic to API when docs available. Remaining criteria: date range filter, multi-select filter, saved filters, export filtered results.</next-steps>
</task>
```

**Next session behavior:** Continue from where this session left off. Read the summary to understand current state. Address the remaining acceptance criteria.

### BLOCKED

The work item could not be started or progressed due to an obstacle.

```xml
<task id="T-3">
  <status>BLOCKED</status>
  <files-modified>false</files-modified>
  <summary>Attempted to optimize database queries but requires schema changes that need DBA approval</summary>
  <obstacles kind="PERMANENT">Database schema changes require DBA review and approval. Cannot proceed without human intervention.</obstacles>
  <next-steps>Escalate to engineering lead for DBA review. Once schema approved, this becomes a standard implementation task (effort: M).</next-steps>
</task>
```

**Next session behavior:** Check if the blocker has been resolved. If yes, attempt the work item. If no, skip and report still blocked.

---

## 4. Blocker Types

### NONE

No blockers. Work completed successfully.

```xml
<obstacles kind="NONE"></obstacles>
```

### TEMPORARY

A transient blocker that is expected to resolve on its own or through routine action.

**Examples:**
- API documentation not yet available (expected soon)
- Dependency version not yet released
- Another team's PR not yet merged
- Test environment temporarily unavailable

```xml
<obstacles kind="TEMPORARY">API v2 documentation not published yet. Expected by end of week per #api-team Slack.</obstacles>
```

**Next session behavior:** Check if the blocker resolved. If yes, proceed. If no, check again or escalate.

### PERMANENT

A blocker that requires human intervention or a decision outside the agent's authority.

**Examples:**
- Architecture decision needed
- Security review required
- Budget approval needed
- DBA review required
- Spec ambiguity that only a human can resolve
- External service access needs provisioning

```xml
<obstacles kind="PERMANENT">Requires architecture decision: should auth tokens be stored in cookies or localStorage? Security implications need human review.</obstacles>
```

**Next session behavior:** Do not attempt this work item until a human has resolved the blocker. Report the blocker in status output.

---

## 5. The files-modified Field

This field indicates whether the session made any code modifications for this work item.

| Value | Meaning |
|-------|---------|
| `true` | Source files, test files, or configuration files were modified |
| `false` | No files were modified (investigation only, or blocked before starting) |

### Why This Matters

- **If `true`:** The next session should check `git log` and `git diff` for what changed
- **If `false`:** The next session knows the codebase is unchanged for this domain
- Helps the convergence monitor understand which iterations produced changes
- Helps the human understand session productivity

---

## 6. Work Queue Handoff Format

The work queue handoff is a separate document (`plan-next-session.md`) that provides the next session with a prioritized list of work items.

### Format

```markdown
# Next Session Work Queue

**Generated:** {timestamp}
**Based on:** implementation tracking, git log, test results, session feedback

---

## WI-1: Complete Dashboard Filtering
- **Category:** feature
- **Size:** M
- **Priority:** high
- **Spec reference:** plan-dashboard.md, Tasks T-4 through T-6
- **Description:** Connect filter logic to API endpoints. The filter UI components exist but are not wired to the backend. API v2 docs should now be available.
- **Files to modify:**
  - `src/dashboard/filters.ts` -- Add API integration
  - `src/api/dashboard-client.ts` -- Add filter endpoint calls
  - `tests/dashboard/filters.test.ts` -- Add filter integration tests
- **Acceptance criteria:**
  - [ ] Date range filter returns correctly filtered results
  - [ ] Multi-select category filter works with 1-5 selections
  - [ ] Filters persist across page navigation
  - [ ] `{TEST_COMMAND}` passes with filter tests included

---

## WI-2: Add API Error Handling
- **Category:** bugfix
- **Size:** S
- **Priority:** high
- **Spec reference:** plan-api.md, Task T-8
- **Description:** API calls currently have no error handling. Network failures crash the UI. Add try/catch with user-facing error messages.
- **Files to modify:**
  - `src/api/client.ts` -- Add error interceptor
  - `src/ui/error-boundary.ts` -- Create error boundary component
  - `tests/api/error-handling.test.ts` -- Add error scenario tests
- **Acceptance criteria:**
  - [ ] Network timeout shows "Connection error" message
  - [ ] 4xx errors show appropriate user message
  - [ ] 5xx errors show "Server error" message with retry button
  - [ ] Error boundary catches unhandled exceptions

---

## WI-3: Implement Data Export
- **Category:** feature
- **Size:** M
- **Priority:** medium
- **Spec reference:** plan-dashboard.md, Tasks T-7 through T-9
- **Description:** Users need to export dashboard data as CSV and PDF. Design exists in spec-dashboard.md R5.
- **Files to modify:**
  - `src/dashboard/export.ts` -- New file, export logic
  - `src/dashboard/export-button.ts` -- New file, UI component
  - `tests/dashboard/export.test.ts` -- New file, export tests
- **Acceptance criteria:**
  - [ ] CSV export includes all visible columns
  - [ ] CSV export respects active filters
  - [ ] PDF export matches dashboard layout
  - [ ] Export works with 10,000+ rows without timeout

---

## WI-4: Resolve Auth Token Storage Decision
- **Category:** investigation
- **Size:** S
- **Priority:** high
- **Spec reference:** plan-auth.md, Task T-10
- **Description:** Blocked in previous session. Needs human decision on cookie vs localStorage for auth tokens. Check if decision has been made.
- **Files to modify:** TBD (depends on decision)
- **Acceptance criteria:**
  - [ ] Decision documented in plan-auth.md
  - [ ] Implementation follows security best practices for chosen approach
  - [ ] Session persistence works across browser tabs

---

## WI-5: Add Performance Benchmarks
- **Category:** test
- **Size:** S
- **Priority:** low
- **Spec reference:** plan-performance.md, Task T-1
- **Description:** Set up performance benchmark infrastructure. Create baseline benchmarks for API response times and page load times.
- **Files to modify:**
  - `tests/benchmarks/api.bench.ts` -- New file
  - `tests/benchmarks/page-load.bench.ts` -- New file
  - `scripts/run-benchmarks.sh` -- New file
- **Acceptance criteria:**
  - [ ] Benchmark script runs without errors
  - [ ] API response time benchmark produces results
  - [ ] Page load benchmark produces results
  - [ ] Results are logged in a parseable format
```

---

## 7. Work Item Fields

### Category

| Value | Description |
|-------|-------------|
| `feature` | New functionality |
| `bugfix` | Fix for a broken behavior |
| `refactor` | Code restructuring without behavior change |
| `test` | Adding or improving tests |
| `investigation` | Research or decision-making (may not produce code) |
| `documentation` | Documentation updates |

### Size

| Value | Meaning | Approximate Time |
|-------|---------|-----------------|
| `S` (Small) | Simple, well-understood change | < 30 minutes |
| `M` (Medium) | Moderate complexity, clear approach | 30-90 minutes |
| `L` (Large) | Complex, may involve multiple files/systems | 90 min - 3 hours |
| `XL` (Extra Large) | Very complex, may need multiple sessions | 3+ hours |

### Priority

| Value | Meaning |
|-------|---------|
| `high` | Blocks other work items or is user-facing |
| `medium` | Important but not blocking |
| `low` | Nice-to-have, can be deferred |

---

## 8. Generating Session Feedback

At the end of each session, the agent generates session feedback by:

### Step 1: Assess Work Items

For each work item from the session's work queue:
1. Check the acceptance criteria -- which are met?
2. Check git log -- what code changes were made for this item?
3. Check test results -- do relevant tests pass?
4. Determine status: DONE (all criteria met), PARTIAL (some met), BLOCKED (none met due to obstacle)

### Step 2: Identify Blockers

For PARTIAL and BLOCKED items:
1. What prevented completion?
2. Is it TEMPORARY (will resolve on its own) or PERMANENT (needs human intervention)?
3. What specifically is needed to unblock?

### Step 3: Write Recommendations

For each work item:
1. If DONE: What should come next? Any follow-up work?
2. If PARTIAL: What exactly remains? Where should the next session pick up?
3. If BLOCKED: What human action is needed? Who should be contacted?

### Step 4: Generate Next Session Work Queue

Based on:
- Remaining work from current session (PARTIAL items)
- New work discovered during this session
- Persistent blockers that may have resolved
- Next priority items from the plan

---

## 9. Consuming Session Feedback

The next session starts by reading the session feedback and work queue:

### Startup Protocol

```markdown
## Session Startup

1. Read `plan-next-session.md` (work queue handoff)
2. Read session feedback from prior session (if available)
3. For each work item:
   a. If from prior session with status DONE -> skip
   b. If from prior session with status PARTIAL -> continue from summary
   c. If from prior session with status BLOCKED -> check if blocker resolved
   d. If new work item -> start fresh
4. Read git state (git log, git status) for current context
5. Read implementation tracking for task status
6. Begin work on highest-priority unblocked item
```

### Blocker Resolution Check

For BLOCKED items from the prior session:

```markdown
Check if the blocker has been resolved:
1. Read the blocker description
2. Check if any new commits address the blocker
3. Check if any new documentation/specs address the blocker
4. If resolved -> proceed with the work item
5. If still blocked -> report still blocked, move to next item
```

---

## 10. Session Feedback in Implementation Tracking

Session feedback integrates with implementation tracking documents:

### After Each Session

```markdown
## Recent Sessions

### Session {date} {time}
**Work items:** WI-1 (DONE), WI-2 (PARTIAL), WI-3 (BLOCKED)
**Summary:** Completed auth module, partially completed dashboard filtering, blocked on DB optimization.
**Net progress:** 2 tasks completed, 1 in progress, 1 blocked.
**Test health:** 45 pass / 3 fail / 2 skip (90% pass rate)

### Session {prior-date} {prior-time}
...
```

### Why Track Sessions

- Provides a historical record of progress
- Helps identify patterns (which areas are repeatedly blocked?)
- Feeds into convergence analysis (are we making forward progress?)
- Helps humans understand what happened without reading all code changes

---

## 11. Multi-Session Workflows

For work that spans multiple sessions, the feedback protocol maintains continuity:

### Session Chain Example

```
Session 1:
  WI-1: Auth -> DONE
  WI-2: Dashboard -> PARTIAL (filtering incomplete)
  WI-3: DB optimization -> BLOCKED (needs DBA)

Session 2:
  WI-2 (continued): Dashboard -> DONE (filtering complete)
  WI-3 (check blocker): DB optimization -> still BLOCKED
  WI-4: Data export -> PARTIAL (CSV done, PDF in progress)

Session 3:
  WI-3 (blocker resolved): DB optimization -> DONE
  WI-4 (continued): Data export -> DONE (PDF complete)
  WI-5: Performance benchmarks -> DONE
```

Each session picks up exactly where the last one left off, with no wasted discovery time.

---

## 12. Integration with Iteration Loops

When running iteration loops, session feedback works slightly differently:

### Single-Session Iteration Loop

Each iteration is a single session. Session feedback is generated at the end of each iteration and consumed at the start of the next:

```
Iteration 1 -> session-feedback-1.xml -> Iteration 2 -> session-feedback-2.xml -> ...
```

In practice, the iteration loop uses implementation tracking as the primary continuity mechanism, with session feedback as supplementary context.

### Multi-Session Projects

For projects that require human-in-the-loop sessions (not automated loops):

```
Human Session 1 (brainstorming, spec review)
  -> session-feedback + plan-next-session.md
    -> Agent Session 2 (implementation iteration loop)
      -> session-feedback + plan-next-session.md
        -> Human Session 3 (review, revision)
          -> session-feedback + plan-next-session.md
            -> Agent Session 4 (implementation iteration loop)
```

The feedback protocol maintains continuity across both human and agent sessions.

---

## 13. Best Practices

### Writing Good Summaries

**Bad:**
```xml
<summary>Worked on auth stuff</summary>
```

**Good:**
```xml
<summary>Implemented login endpoint (POST /api/auth/login) with bcrypt password verification, JWT token generation, and session creation. Added 8 unit tests covering happy path, invalid credentials, locked accounts, and token expiry. All tests passing.</summary>
```

### Writing Good Recommendations

**Bad:**
```xml
<next-steps>Keep going</next-steps>
```

**Good:**
```xml
<next-steps>Auth login is complete. Next: implement logout endpoint (spec-auth.md R3), then session refresh (spec-auth.md R4). The session middleware (src/middleware/auth.ts) is already in place and can be reused for logout. Token refresh will need a new database table -- see plan-auth.md T-7 for schema.</next-steps>
```

### Writing Good Blocker Descriptions

**Bad:**
```xml
<obstacles kind="PERMANENT">Can't do it</obstacles>
```

**Good:**
```xml
<obstacles kind="PERMANENT">The spec requires OAuth2 integration with a third-party provider, but no provider credentials have been configured. Need: (1) OAuth client ID and secret for the provider, (2) redirect URL configuration, (3) decision on which OAuth scopes to request. Contact: engineering lead for provider setup.</obstacles>
```

---

## 14. Session Feedback Schema Summary

```
<session-report>
  <task id="T-{N}">                    # 1..N work items
    <status>DONE|PARTIAL|BLOCKED</status>
    <files-modified>true|false</files-modified>
    <summary>{what was accomplished}</summary>
    <obstacles kind="NONE|TEMPORARY|PERMANENT">
      {blocker description if any}
    </obstacles>
    <next-steps>{next steps}</next-steps>
  </task>
</session-report>
```

### Rules

1. Every task from the session's queue must appear in the report
2. Status must be one of exactly three values: DONE, PARTIAL, BLOCKED
3. Obstacle kind must be one of exactly three values: NONE, TEMPORARY, PERMANENT
4. Summary must be specific and actionable, not vague
5. Next-steps must tell the following session exactly what to do
6. files-modified must accurately reflect whether source files were changed
7. Tasks are identified by their T- ID for cross-session tracking
