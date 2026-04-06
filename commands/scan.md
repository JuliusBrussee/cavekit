---
name: ck-scan
description: Compare what was built against what was intended
---

> **Note:** `/bp:gap-analysis`, `/ck:gap-analysis`, `/bp:scan` are deprecated aliases. Use `/ck:scan` instead.

# Cavekit Gap Analysis — Built vs. Intended

You are performing a gap analysis: comparing what was actually built (implementation tracking, code, test results) against what was intended (kits, acceptance criteria). This identifies where kits, plans, or validation fell short and feeds into revision.

Before dispatching any agent:

1. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" summary` and print that exact line once.
2. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" model reasoning` and treat the result as `REASONING_MODEL`.

Dispatch a `ck:surveyor` agent with `model: "{REASONING_MODEL}"` and the following instructions. If the agent tool is unavailable, execute the instructions directly.

## Agent Instructions for surveyor

### Phase 1: Read Kits (Intended)

1. Read `context/kits/cavekit-overview.md` to get the full domain map
2. For each domain, read `context/kits/cavekit-{domain}.md`
3. Catalog every requirement and its acceptance criteria into a checklist:
   - Requirement ID (R1, R2, etc.)
   - Requirement description
   - Each acceptance criterion
   - Source cavekit file

### Phase 2: Read Implementation (Built)

1. Read all impl tracking files in `context/impl/`
2. Read `context/plans/plan-build-site.md` for tier/progress state
3. Read `context/plans/plan-known-issues.md` for documented issues
4. Optionally run the test suite and capture results

### Phase 3: Compare

For each cavekit requirement and acceptance criterion, classify it:

| Status | Definition |
|--------|-----------|
| **COMPLETE** | Acceptance criteria met, tests pass, code exists |
| **PARTIAL** | Some criteria met, others missing or failing |
| **MISSING** | Not implemented at all |
| **OVER-BUILT** | Implementation goes beyond what the cavekit requires (scope creep) |
| **UNTESTABLE** | Acceptance criteria cannot be automatically validated |

### Phase 4: Identify Revision Targets

For each gap (PARTIAL, MISSING, OVER-BUILT, UNTESTABLE), determine the root cause:

| Root Cause | Description | Fix Target |
|-----------|-------------|------------|
| **Cavekit gap** | Requirement was ambiguous or missing detail | Update cavekit |
| **Plan gap** | Cavekit was clear but plan didn't cover it | Update plan |
| **Implementation gap** | Plan covered it but implementation is incomplete | Continue implementing |
| **Validation gap** | Built but no test verifies it | Add tests |
| **Scope creep** | Built without a cavekit requirement | Either add cavekit or remove code |

### Phase 5: Generate Report

```markdown
## Gap Analysis Report

### Summary
| Status | Count | Percentage |
|--------|-------|-----------|
| COMPLETE | {n} | {%} |
| PARTIAL | {n} | {%} |
| MISSING | {n} | {%} |
| OVER-BUILT | {n} | {%} |
| UNTESTABLE | {n} | {%} |

### Overall Coverage: {complete + partial} / {total requirements}

### Detailed Findings

#### Complete
| Requirement | Cavekit | Evidence |
|------------|------|----------|
| R1: {name} | cavekit-{domain}.md | {test file or impl reference} |

#### Partial
| Requirement | Cavekit | Met | Missing | Root Cause |
|------------|------|-----|---------|-----------|
| R2: {name} | cavekit-{domain}.md | {criteria met} | {criteria missing} | {cavekit/plan/impl/validation gap} |

#### Missing
| Requirement | Cavekit | Root Cause | Suggested Action |
|------------|------|-----------|-----------------|
| R3: {name} | cavekit-{domain}.md | {root cause} | {action} |

#### Over-Built
| Feature | Files | Cavekit Coverage | Suggested Action |
|---------|-------|--------------|-----------------|
| {feature} | {files} | No cavekit | Add cavekit or remove |

#### Untestable
| Requirement | Cavekit | Why Untestable | Suggested Action |
|------------|------|---------------|-----------------|
| R4: {name} | cavekit-{domain}.md | {reason} | {rewrite criteria / add human review gate} |

### Revision Targets
| Priority | Target File | Change Needed | Affected Requirements |
|----------|------------|--------------|----------------------|
| P0 | cavekit-{domain}.md | {what to update} | R{n}, R{n} |
| P1 | plan-{domain}.md | {what to update} | R{n} |

### Recommended Next Steps
1. Run `/ck:revise` to trace gaps into context files
2. {Specific cavekit updates needed}
3. {Specific plan updates needed}
4. {Implementation work remaining}
```

Present this report to the user when complete.
