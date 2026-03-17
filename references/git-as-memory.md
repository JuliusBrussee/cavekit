# Git as Agent Working Memory Reference

How git serves as persistent working memory for AI agents across iteration loops. Covers the feature table, commit patterns, iteration start patterns, and branch/worktree strategies.

---

## 1. Overview

For AI agents, git serves a purpose beyond version control -- it functions as **durable state storage** that persists across iteration boundaries. Each iteration loop pass starts fresh with no conversation history, but the full state of the project is preserved in git. This makes git the primary mechanism for cross-iteration continuity.

**Key Insight:** Git is local and fast. Agents can commit freely without hitting servers. This is why the fundamental rule is:

> **Commit often, push only when a human says so.**

---

## 2. Git Feature Table

How each git feature serves the agent:

| Git Feature | Agent Use | When |
|-------------|-----------|------|
| **Local commits** | Progress cookies between iterations | After each meaningful change |
| **Commit history** (`git log`) | Understand what was done in prior iterations | Start of each iteration |
| **Diff** (`git diff`) | See what changed since last commit | During work, before commits |
| **Branches** | Isolate experimental work | When trying uncertain approaches |
| **Worktrees** | Parallel agent isolation | During agent team work |
| **Status** (`git status`) | Check what is modified/untracked | Start of each iteration, before commits |
| **Stash** (`git stash`) | Temporarily set aside work | When switching between tasks |
| **Tags** | Mark significant milestones | End of successful iteration runs |
| **Blame** (`git blame`) | Understand who/what changed a file | During debugging and revision |

---

## 3. The "Commit Frequently, Never Push" Pattern

### Why Commit Frequently

Each commit is a **progress cookie** -- a checkpoint that:
- Records what was accomplished
- Can be read by the next iteration (`git log`)
- Can be compared against (`git diff HEAD~1`)
- Can be reverted if an approach fails
- Provides granular history for revision

### When to Commit

| Event | Commit? | Message Pattern |
|-------|---------|----------------|
| Task completed | Yes | `feat: implement T-3 data models` |
| Tests passing | Yes | `test: add unit tests for auth module` |
| Bug fixed | Yes | `fix: resolve null check in login flow` |
| Refactor complete | Yes | `refactor: extract shared validation logic` |
| Work in progress (switching tasks) | Yes | `wip: T-5 partial implementation` |
| Before trying uncertain approach | Yes | `checkpoint: before experimental approach` |
| After successful merge | Yes | `merge: integrate auth-teammate work` |

### Why Never Push

Pushing to a remote repository:
- Is irreversible (force-push is dangerous and may not be permitted)
- Affects other developers and CI systems
- May trigger deployment pipelines
- Is not needed for agent working memory (git is local)

**The only time to push is when a human explicitly requests it**, typically after reviewing the work and approving it.

### Commit Message Best Practices for Agents

```markdown
## Commit Message Format

<type>: <short description>

<optional body explaining WHY, not just WHAT>

Types:
- feat: New feature or capability
- fix: Bug fix
- test: Adding or updating tests
- refactor: Code restructuring without behavior change
- docs: Documentation updates
- wip: Work in progress (checkpoint)
- merge: Merge of agent team work
- checkpoint: Pre-experiment savepoint
```

---

## 4. Iteration Start Pattern

Every iteration begins by reading git state to understand the current position. This is how agents maintain continuity without conversation history.

### The Pattern

```markdown
## Start of Each Iteration

1. Read git status:
   git status
   -> Understand what files are modified, staged, untracked

2. Read recent git history:
   git log --oneline -20
   -> Understand what was accomplished in prior iterations

3. Read recent changes:
   git diff HEAD~3 --stat
   -> Understand the scope of recent work

4. Read implementation tracking:
   Read context/impl/impl-*.md
   -> Understand task status, issues, dead ends

5. Based on all of the above, determine:
   - What is the current state?
   - What was the last thing done?
   - What should be done next?
   - Are there any unresolved issues?
```

### What Each Git Command Tells the Agent

| Command | What It Reveals |
|---------|----------------|
| `git status` | Files currently modified but uncommitted; untracked files; staged changes |
| `git log --oneline -20` | Last 20 commits with short descriptions; chronological progress |
| `git log --oneline --since="4 hours ago"` | Recent session work |
| `git diff` | Unstaged changes (work in progress) |
| `git diff --cached` | Staged changes (ready to commit) |
| `git diff HEAD~1` | Changes in the last commit |
| `git diff HEAD~3 --stat` | Summary of changes over last 3 commits |
| `git diff main...HEAD` | All changes on this branch vs main |

### Why This Pattern Works

Without conversation history, the agent has no memory of what it did in the previous iteration. But by reading git state:
- Commit messages explain what was done and why
- Diffs show what changed
- Status shows what is in progress
- Implementation tracking provides structured context

Together, these give the agent a complete picture of project state without any conversation memory.

---

## 5. Branches for Experiments

When an agent wants to try an uncertain approach, it should create a branch:

### Experimental Branch Pattern

```bash
# Before trying something uncertain
git checkout -b experiment/{description}

# Try the approach
# ... work ...

# If it works:
git checkout main
git merge experiment/{description}
git branch -d experiment/{description}

# If it fails:
git checkout main
git branch -D experiment/{description}
# Document the dead end in implementation tracking
```

### When to Branch

| Situation | Branch? | Rationale |
|-----------|---------|-----------|
| Trying a new algorithm | Yes | Easy to revert if it fails |
| Refactoring shared code | Yes | Protects main from breakage |
| Exploratory research | Yes | Keeps main clean |
| Normal task implementation | No | Main branch is fine |
| Bug fix with known solution | No | Low risk, commit directly |

### Dead End Documentation

When an experimental branch fails, document it in implementation tracking:

```markdown
## Dead Ends & Failed Approaches

### Approach: {description}
- **Branch:** experiment/{description}
- **What was tried:** {description of approach}
- **Why it failed:** {root cause}
- **Time spent:** {duration}
- **Lesson:** {what to do differently}
- **Do NOT retry this approach.**
```

This prevents future iterations from wasting time on the same failed approach.

---

## 6. Worktrees for Parallel Isolation

Git worktrees create completely isolated working copies, each with their own working directory but sharing the same git repository.

### When to Use Worktrees

| Scenario | Use Worktree |
|----------|-------------|
| Agent team with multiple teammates | Yes, one per teammate |
| Single agent, single task | No |
| Single agent, multiple independent tasks | Maybe, if tasks touch different files |
| Experimentation alongside main work | Yes, keeps main clean |

### Worktree Lifecycle

```bash
# 1. Create worktree
git worktree add ./worktrees/{name} -b feat/impl/{name}

# 2. Work in the worktree
cd ./worktrees/{name}
# ... implement ...
git add .
git commit -m "feat: implement {feature}"

# 3. Merge back to main
cd /project-root
git checkout main
git merge feat/impl/{name} --no-ff

# 4. Validate
{BUILD_COMMAND}
{TEST_COMMAND}

# 5. Clean up
git worktree remove ./worktrees/{name}
git branch -d feat/impl/{name}
```

### Worktree Isolation Properties

| Property | Behavior |
|----------|----------|
| Working directory | Completely separate per worktree |
| Staged changes | Separate per worktree |
| Branch | Each worktree is on its own branch |
| Commits | Committed to the worktree's branch |
| Git objects | Shared (same .git repository) |
| Refs | Shared (branches visible across worktrees) |
| Hooks | Shared |

---

## 7. Git State as Context for Agents

### What to Include in Prompts

Prompts should instruct agents to read git state at the start of each iteration:

```markdown
## Context Gathering

Before starting work, read the current state:

1. `git status` -- What files are modified/untracked?
2. `git log --oneline -10` -- What was done recently?
3. Read `context/impl/impl-*.md` -- What is the task status?

Based on this context, determine the highest-priority unblocked task
and proceed with implementation.
```

### What NOT to Do

- Do NOT rely on conversation history between iterations
- Do NOT assume files are in a particular state without checking
- Do NOT assume prior iterations completed successfully
- Do NOT skip the git state reading step

---

## 8. Git Patterns for Specific Scenarios

### Rolling Back a Failed Change

```bash
# Revert the last commit (creates a new commit)
git revert HEAD

# Or reset to before the last commit (destructive)
git reset --hard HEAD~1
```

### Finding When Something Broke

```bash
# Binary search through commits
git bisect start
git bisect bad HEAD
git bisect good {known-good-commit}
# Git will check out commits for testing
{TEST_COMMAND}
git bisect good  # or git bisect bad
# Repeat until the breaking commit is found
git bisect reset
```

### Understanding a File's History

```bash
# Who changed what and when
git log --follow -p -- {file}

# Line-by-line attribution
git blame {file}
```

### Comparing Branches

```bash
# What is in feature branch but not in main
git log main..feat/impl/{name} --oneline

# Full diff between branches
git diff main...feat/impl/{name}

# Files changed between branches
git diff main...feat/impl/{name} --stat
```

---

## 9. Progress Tracking via Git

Git history provides natural progress tracking:

### Iteration Progress

```bash
# Count commits per iteration (if tagged)
git log --oneline v1..v2 | wc -l

# Lines changed per iteration
git diff v1..v2 --stat | tail -1

# Files touched per iteration
git diff v1..v2 --name-only | wc -l
```

### Convergence Detection

```bash
# Compare change volume across iterations
# Iteration N:
git diff HEAD~1 --stat | tail -1  # e.g., "20 files changed, 450 insertions, 120 deletions"

# Iteration N+1:
git diff HEAD~1 --stat | tail -1  # e.g., "12 files changed, 180 insertions, 60 deletions"

# Iteration N+2:
git diff HEAD~1 --stat | tail -1  # e.g., "5 files changed, 45 insertions, 20 deletions"

# Decreasing numbers = convergence
```

### Revision Support

```bash
# Find commits that were manual fixes (not from iteration loop)
git log --oneline --author="human" --since="1 week ago"

# Analyze what a manual fix changed
git show {commit-hash} --stat
git show {commit-hash} -p
```

---

## 10. Git Configuration for SDD Projects

### Recommended .gitignore Additions

```gitignore
# Worktrees directory (each worktree has its own .git link)
worktrees/

# Agent session artifacts
.claude/

# Build artifacts (project-specific)
dist/
build/
node_modules/
```

### Recommended Git Practices

| Practice | Rationale |
|----------|-----------|
| Short-lived branches | Reduce merge conflict risk |
| Frequent commits | More granular progress cookies |
| Descriptive commit messages | Better context for next iteration |
| Tags for milestones | Easy to compare before/after |
| No force-push | Preserve history integrity |
| No push without human approval | Prevent accidental deployments |

---

## 11. Summary

Git serves as the complete working memory system for AI agents in SDD:

1. **Commits are progress cookies** -- each commit marks a checkpoint the next iteration can read
2. **Commit messages are memory** -- they explain what was done and why
3. **Diffs show changes** -- agents understand what happened by reading diffs
4. **Branches isolate experiments** -- failed approaches can be cleanly abandoned
5. **Worktrees isolate agents** -- parallel work without file conflicts
6. **Status shows current state** -- agents know what is in progress
7. **History supports revision** -- tracing bugs back to their source

The fundamental rule remains:

> **Commit often, push only when a human says so.**

Every iteration starts by reading git state. Every meaningful change results in a commit. This creates a complete, persistent record that bridges the gap between stateless iteration loop passes.
