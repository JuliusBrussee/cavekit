# Multi-Repo Strategy Reference

Shared base context with framework-specific repositories. Covers the 3-tier hierarchy, git submodule integration, and cross-framework spec sharing.

---

## 1. Overview

When evaluating multiple frameworks or maintaining shared specifications across different technology implementations, SDD uses a **shared base context** with framework-specific repositories connected via git submodules.

This enables:
- Same specs driving different implementations
- Apples-to-apples framework comparison
- Spec updates propagating to all implementations
- Independent divergence where needed

---

## 2. The 3-Tier Context Hierarchy

### Tier 1: Shared Context -- What IS (Reference Materials)

Reference materials that describe the current state of the system or the source of truth.

```
shared-context/
+-- reference/
    +-- ref-apis.md           # API documentation
    +-- ref-data-models.md    # Data model documentation
    +-- ref-ui-components.md  # UI component inventory
    +-- ref-library.md        # Library/framework docs
    +-- ref-architecture.md   # Architecture overview
```

**Characteristics:**
- Framework-agnostic
- Describes what currently exists or what the source material says
- Read-only for implementation agents (only updated by spec agents)
- Shared across all implementations

### Tier 2: Shared Context -- What MUST BE (Specifications)

Implementation-agnostic specifications that define what the new system must do.

```
shared-context/
+-- specs/
|   +-- blueprint-overview.md           # Index of all specs
|   +-- spec-features.md           # Feature requirements
|   +-- spec-ui-requirements.md    # UI/UX requirements
|   +-- spec-data.md               # Data model specs
|   +-- spec-auth.md               # Authentication specs
|   +-- spec-api.md                # API contract specs
+-- prompts/
    +-- 001-generate-refs.md       # Shared prompt: generate refs
    +-- 002-generate-specs.md      # Shared prompt: generate specs
    +-- 003-validate-specs.md      # Shared prompt: validate specs
```

**Characteristics:**
- Framework-agnostic (describes WHAT, not HOW)
- Shared across all implementations
- Updated through revision from any implementation
- Contains shared prompts for spec-level phases

### Tier 3: Application Context (Framework-Specific)

Plans, implementation tracking, and framework-specific prompts for each implementation.

```
{framework}-prototype/
+-- context/
|   +-- plans/
|   |   +-- plan-build-site.md  # Feature dependencies for this framework
|   |   +-- plan-known-issues.md      # Known issues in this implementation
|   |   +-- plan-auth.md              # How to implement auth in {FRAMEWORK}
|   |   +-- plan-ui.md                # How to implement UI in {FRAMEWORK}
|   +-- impl/
|   |   +-- impl-all.md              # Implementation tracking
|   +-- prompts/
|   |   +-- 004-create-plans.md      # Framework-specific prompt
|   |   +-- 005-implement.md         # Framework-specific prompt
|   |   +-- 006-update-specs.md      # Revision prompt
|   +-- research/
|       +-- research-{framework}.md  # Framework research findings
+-- shared-context/                   # <-- git submodule
+-- src/                              # Framework-specific source code
+-- tests/                            # Framework-specific tests
```

**Characteristics:**
- Framework-specific (describes HOW for this particular stack)
- Contains plans, implementation tracking, research
- Has its own prompts for plan/implement/iterate phases
- Links to shared context via git submodule

---

## 3. Repository Structure

### The Shared Context Repository

```
shared-context-repo/
+-- reference/
|   +-- ref-apis.md
|   +-- ref-data-models.md
|   +-- ref-ui-components.md
|   +-- ref-library.md
+-- specs/
|   +-- blueprint-overview.md
|   +-- spec-features.md
|   +-- spec-ui-requirements.md
|   +-- spec-data.md
|   +-- spec-auth.md
|   +-- spec-api.md
+-- prompts/
|   +-- 001-generate-refs.md
|   +-- 002-generate-specs.md
|   +-- 003-validate-specs.md
+-- CLAUDE.md
```

### A Framework-Specific Repository

```
react-prototype/
+-- shared-context/           # git submodule -> shared-context-repo
+-- context/
|   +-- plans/
|   +-- impl/
|   +-- prompts/
|   +-- research/
+-- src/
+-- tests/
+-- package.json
+-- CLAUDE.md
```

### Another Framework-Specific Repository

```
tauri-prototype/
+-- shared-context/           # git submodule -> shared-context-repo (same repo!)
+-- context/
|   +-- plans/
|   +-- impl/
|   +-- prompts/
|   +-- research/
+-- src-tauri/
+-- src/
+-- tests/
+-- Cargo.toml
+-- CLAUDE.md
```

Both framework repositories point to the **same** shared context repository. Changes to specs propagate to both.

---

## 4. Git Submodule Integration

### Setting Up Submodules

```bash
# In the framework-specific repository
git submodule add {shared-context-repo-url} shared-context
git commit -m "Add shared context submodule"
```

### Updating Shared Context

```bash
# Pull latest specs/reference from shared context
cd shared-context
git pull origin main
cd ..
git add shared-context
git commit -m "Update shared context to latest specs"
```

### Revision Across Repos

When an implementation discovers a spec gap:

```bash
# 1. In the framework repo, identify the spec gap
# 2. Switch to the shared context submodule
cd shared-context

# 3. Create a branch for the spec update
git checkout -b fix/spec-gap-{description}

# 4. Update the spec
# ... edit spec files ...
git add .
git commit -m "fix: add missing requirement for {description}"

# 5. Push the spec update (or create PR)
git push origin fix/spec-gap-{description}

# 6. Back in the framework repo, update the submodule reference
cd ..
git add shared-context
git commit -m "Update shared context with spec fix"
```

### All Framework Repos Get the Update

```bash
# In each framework-specific repo
cd shared-context
git pull origin main  # Gets the spec fix
cd ..
git add shared-context
git commit -m "Update shared context with latest spec fixes"

# Re-run iteration loop -- the fix should emerge from updated specs
```

---

## 5. Prompt Organization in Multi-Repo

### Shared Prompts (in shared-context)

Prompts that operate on shared artifacts (refs, specs):

| Prompt | Phase | Input | Output |
|--------|-------|-------|--------|
| `001-generate-refs.md` | Spec (prep) | Old app/source material | `reference/` |
| `002-generate-specs.md` | Spec | Feature scope + reference | `specs/` |
| `003-validate-specs.md` | Spec (verify) | Reference + specs | Validation report |

### Framework-Specific Prompts (in context/)

Prompts that operate on framework-specific artifacts:

| Prompt | Phase | Input | Output |
|--------|-------|-------|--------|
| `004-create-plans.md` | Plan | Specs + framework research | `plans/` |
| `005-implement.md` | Implement | Plans + specs | `src/`, `tests/`, `impl/` |
| `006-update-specs.md` | Iterate | Working prototype | Updated specs (revision) |

### Runtime Variables in Multi-Repo Prompts

Shared prompts use no framework-specific references. Framework-specific prompts use runtime variables:

```markdown
## Framework-Specific Configuration

- **Framework:** {FRAMEWORK}
- **Build:** {BUILD_COMMAND}
- **Test:** {TEST_COMMAND}
- **Dev server:** {DEV_COMMAND}

Read `context/research/research-{FRAMEWORK}.md` for framework patterns.
```

---

## 6. Benefits of Multi-Repo Strategy

### Same Specs, Different Stacks

The most powerful benefit: identical specifications drive completely different implementations.

```
shared-context/blueprints/spec-auth.md
    |
    +-> react-prototype/context/plans/plan-auth.md (React + NextAuth)
    |   +-> React implementation with NextAuth
    |
    +-> django-prototype/context/plans/plan-auth.md (Django + django-allauth)
    |   +-> Django implementation with django-allauth
    |
    +-> go-prototype/context/plans/plan-auth.md (Go + gorilla/sessions)
        +-> Go implementation with gorilla/sessions
```

All three implementations satisfy the same spec requirements. The plans differ (framework-specific HOW), but the specs (WHAT) are identical.

### Apples-to-Apples Comparison

When evaluating frameworks, multi-repo strategy ensures fair comparison:

| Dimension | How It Helps |
|-----------|-------------|
| Feature parity | Same specs = same features in each prototype |
| Quality comparison | Same validation gates applied to each |
| Effort comparison | Implementation tracking shows effort per framework |
| Complexity comparison | Plan files reveal architectural differences |
| Performance comparison | Same benchmarks run against each prototype |

### Spec Updates Propagate

When a spec bug is found in one implementation:
1. Fix the spec in shared context
2. Update submodule in all framework repos
3. Re-run iteration loops
4. All implementations get the fix

This prevents spec drift between implementations.

### Independent Divergence

Despite sharing specs, each implementation can:
- Add framework-specific features not in shared specs
- Use different architectural approaches
- Have different test strategies
- Diverge in plans without affecting shared specs

---

## 7. Generic Multi-Repo Example

### Scenario: Web Application with Multiple Framework Candidates

**Shared Context:**
```
web-app-specs/
+-- reference/
|   +-- ref-api-contracts.md     # REST API documentation
|   +-- ref-data-models.md       # Database schema documentation
|   +-- ref-user-flows.md        # User journey documentation
+-- specs/
|   +-- blueprint-overview.md         # All features with acceptance criteria
|   +-- spec-auth.md             # Authentication: login, register, sessions
|   +-- spec-dashboard.md        # Dashboard: data display, filtering, export
|   +-- spec-settings.md         # Settings: profile, preferences, notifications
|   +-- spec-api.md              # API: endpoints, contracts, error handling
+-- prompts/
    +-- 001-generate-refs.md     # Extract refs from old app
    +-- 002-generate-specs.md    # Generate specs from refs
```

**Framework A Implementation:**
```
framework-a-prototype/
+-- shared-context/              # submodule -> web-app-specs
+-- context/
|   +-- plans/
|   |   +-- plan-auth.md        # Auth via Framework A patterns
|   |   +-- plan-dashboard.md   # Dashboard via Framework A patterns
|   +-- impl/
|   +-- prompts/
|       +-- 004-create-plans.md
|       +-- 005-implement.md
+-- src/
+-- tests/
```

**Framework B Implementation:**
```
framework-b-prototype/
+-- shared-context/              # submodule -> web-app-specs (same!)
+-- context/
|   +-- plans/
|   |   +-- plan-auth.md        # Auth via Framework B patterns
|   |   +-- plan-dashboard.md   # Dashboard via Framework B patterns
|   +-- impl/
|   +-- prompts/
|       +-- 004-create-plans.md
|       +-- 005-implement.md
+-- src/
+-- tests/
```

### How the Evaluation Works

1. **Generate shared specs** (run once, shared across both):
   ```bash
   cd web-app-specs
   # Run iteration loop on prompts 001-002
   ```

2. **Generate framework-specific plans** (run in each prototype):
   ```bash
   cd framework-a-prototype
   # Run iteration loop on prompt 004

   cd framework-b-prototype
   # Run iteration loop on prompt 004
   ```

3. **Implement** (run in each prototype):
   ```bash
   cd framework-a-prototype
   # Run iteration loop on prompt 005

   cd framework-b-prototype
   # Run iteration loop on prompt 005
   ```

4. **Compare results:**
   - Same acceptance criteria, different implementations
   - Compare: code complexity, test coverage, performance, developer experience
   - Decision is data-driven, not opinion-driven

---

## 8. CLAUDE.md in Multi-Repo

### Shared Context CLAUDE.md

```markdown
# Shared Context

This directory contains framework-agnostic reference materials and specifications.

## Directory Structure
- `reference/` -- Source of truth documentation (What IS)
- `specs/` -- Implementation-agnostic specifications (What MUST BE)
- `prompts/` -- Shared DABI prompts (spec-level phases only)

## Conventions
- Specs describe WHAT, never HOW
- Every requirement has testable acceptance criteria
- Cross-reference related specs
- Do NOT add framework-specific content here
```

### Framework Repo CLAUDE.md

```markdown
# {FRAMEWORK} Prototype

Implementation of shared specs using {FRAMEWORK}.

## Shared Context
Shared specs and reference materials are in `shared-context/` (git submodule).
Run `git submodule update --init` if missing.

## Project-Specific
- `context/plans/` -- {FRAMEWORK}-specific implementation plans
- `context/impl/` -- Implementation tracking
- `context/prompts/` -- {FRAMEWORK}-specific prompts
- `context/research/` -- {FRAMEWORK} research findings

## Commands
- Build: {BUILD_COMMAND}
- Test: {TEST_COMMAND}
- Dev: {DEV_COMMAND}
```

---

## 9. When to Use Multi-Repo Strategy

### Use When

| Scenario | Why Multi-Repo |
|----------|---------------|
| Framework evaluation | Fair apples-to-apples comparison |
| Migration projects | Old and new coexist with shared specs |
| Multi-platform | Same app on web, mobile, desktop with shared specs |
| Team specialization | Different teams own different implementations |
| Client customization | Shared core specs with client-specific implementations |

### Do NOT Use When

| Scenario | Why Not |
|----------|---------|
| Single framework, single repo | Overhead not justified |
| Small project (<50 files) | Single context directory is sufficient |
| No shared specs | Nothing to share across repos |
| Tight coupling between implementations | Submodule updates would break things |

---

## 10. Submodule Best Practices

### Keep Shared Context Lean

The shared context should contain only:
- Reference materials (Tier 1)
- Specifications (Tier 2)
- Shared prompts (spec-level only)

It should NOT contain:
- Framework-specific code
- Plans or implementation tracking
- Framework-specific prompts
- Build artifacts

### Pin Submodule Versions

Each framework repo pins to a specific commit of shared context. This prevents unexpected changes from breaking implementations:

```bash
# Pin to specific version
cd shared-context
git checkout {commit-hash}
cd ..
git add shared-context
git commit -m "Pin shared context to v2.1 specs"
```

### Update Deliberately

Update the submodule reference explicitly, not automatically:

```bash
# Update to latest
cd shared-context
git pull origin main
cd ..
git add shared-context
git commit -m "Update shared context: added auth session specs"

# Verify nothing broke
{BUILD_COMMAND}
{TEST_COMMAND}
```

### Document Submodule Relationship

Always document the submodule relationship in the README or CLAUDE.md so agents know where to find shared artifacts:

```markdown
## Shared Context

This project uses shared specifications from `shared-context/`.
To initialize: `git submodule update --init --recursive`
To update: `cd shared-context && git pull origin main && cd ..`
```

---

## 11. Revision in Multi-Repo

When a bug is found in one implementation:

### If the Bug Is Spec-Level

1. The spec is incomplete or ambiguous
2. Fix the spec in shared context
3. Update submodule in ALL framework repos
4. Re-run iteration loops in ALL implementations
5. All implementations get the fix

### If the Bug Is Plan-Level

1. The plan incorrectly translates the spec for this framework
2. Fix the plan in the framework-specific repo only
3. No shared context change needed
4. Only this implementation is affected

### If the Bug Is Code-Level

1. Implementation error, spec and plan are correct
2. Fix the code in the framework-specific repo
3. No shared context change needed
4. Consider if the plan needs a clearer explanation

### Decision Matrix

```
Bug found in Framework A implementation
  |
  +-> Is the spec missing a requirement?
  |     YES -> Fix shared-context/blueprints/ -> update ALL repos
  |     NO  -> continue
  |
  +-> Is the plan wrong for this framework?
  |     YES -> Fix context/plans/ -> update only this repo
  |     NO  -> continue
  |
  +-> Is the code wrong?
        YES -> Fix the code, consider plan clarity
```

---

## 12. Summary

The multi-repo strategy enables:

1. **Shared specifications** across multiple implementations
2. **Fair framework comparison** using identical acceptance criteria
3. **Spec propagation** -- a fix in one place reaches all implementations
4. **Independent plans** -- each framework has its own implementation approach
5. **Git submodules** as the connection mechanism
6. **3-tier hierarchy** separating What IS, What MUST BE, and How To Build
7. **Revision across repos** -- spec bugs found anywhere fix everywhere
