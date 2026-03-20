#!/bin/bash

# Blueprint Build Setup Script
# Archives old cycle, reads frontier, starts Ralph Loop.
# Optionally configures Codex MCP for peer review.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

FILTER=""
PEER_REVIEW=false
ABANDON=false
MAX_ITERATIONS=20
COMPLETION_PROMISE="BLUEPRINT COMPLETE"
CODEX_MODEL="gpt-5.4"
REVIEW_INTERVAL=2

while [[ $# -gt 0 ]]; do
  case $1 in
    -h|--help)
      cat << 'HELP_EOF'
Blueprint Build — Run the implementation loop

USAGE:
  /blueprint build [OPTIONS]

OPTIONS:
  --filter <pattern>             Scope to blueprints/build site matching pattern
  --peer-review                  Add Codex (GPT-5.4) peer review
  --codex-model <model>          Codex model (default: gpt-5.4)
  --review-interval <n>          Review every Nth iteration (default: 2)
  --max-iterations <n>           Max iterations (default: 20)
  --abandon                      Remove worktree and branch for the filtered build
  --completion-promise '<text>'  Completion phrase (default: "BLUEPRINT COMPLETE")
  -h, --help                     Show this help

EXAMPLES:
  /blueprint build
  /blueprint build --filter v2
  /blueprint build --peer-review
  /blueprint build --peer-review --max-iterations 30
  /blueprint build --filter v2 --abandon
HELP_EOF
      exit 0
      ;;
    --filter)
      [[ -z "${2:-}" ]] && { echo "❌ --filter requires a pattern" >&2; exit 1; }
      FILTER="$2"
      shift 2
      ;;
    --peer-review)
      PEER_REVIEW=true
      shift
      ;;
    --abandon)
      ABANDON=true
      shift
      ;;
    --codex-model)
      [[ -z "${2:-}" ]] && { echo "❌ --codex-model requires a model name" >&2; exit 1; }
      CODEX_MODEL="$2"
      shift 2
      ;;
    --review-interval)
      [[ -z "${2:-}" ]] && { echo "❌ --review-interval requires a number" >&2; exit 1; }
      REVIEW_INTERVAL="$2"
      shift 2
      ;;
    --max-iterations)
      [[ -z "${2:-}" ]] && { echo "❌ --max-iterations requires a number" >&2; exit 1; }
      [[ ! "$2" =~ ^[0-9]+$ ]] && { echo "❌ --max-iterations must be a positive integer" >&2; exit 1; }
      MAX_ITERATIONS="$2"
      shift 2
      ;;
    --completion-promise)
      [[ -z "${2:-}" ]] && { echo "❌ --completion-promise requires text" >&2; exit 1; }
      COMPLETION_PROMISE="$2"
      shift 2
      ;;
    *)
      echo "❌ Unexpected argument: $1" >&2
      exit 1
      ;;
  esac
done

# ─── Create worktree if not already in one ──────────────────────────────────

PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
PROJECT_NAME="$(basename "$PROJECT_ROOT")"

# Detect if we're already in a worktree (blueprint --monitor creates these)
IS_WORKTREE=false
GIT_COMMON_DIR="$(git rev-parse --git-common-dir 2>/dev/null || true)"
GIT_DIR="$(git rev-parse --git-dir 2>/dev/null || true)"
if [[ -n "$GIT_COMMON_DIR" && -n "$GIT_DIR" && "$GIT_COMMON_DIR" != "$GIT_DIR" ]]; then
  IS_WORKTREE=true
fi

if [[ "$IS_WORKTREE" == "false" ]]; then
  # Clean any stale ralph-loop state from the main project dir
  # This prevents the stop hook from hijacking non-Blueprint conversations
  rm -f "$PROJECT_ROOT/.claude/ralph-loop.local.md"

  # Derive worktree name from filter or build site
  WT_NAME="${FILTER:-build}"
  WT_PATH="${PROJECT_ROOT}/../${PROJECT_NAME}-blueprint-${WT_NAME}"
  BRANCH_NAME="blueprint/${WT_NAME}"

  # ─── Abandon mode (R3: Failure Recovery — option b) ──────────────────
  if [[ "$ABANDON" == "true" ]]; then
    if [[ -d "$WT_PATH" ]]; then
      echo "🗑️  Abandoning worktree: $WT_PATH"
      git worktree remove --force "$WT_PATH" 2>/dev/null || rm -rf "$WT_PATH"
      git worktree prune 2>/dev/null || true
      # Delete branch if it exists and has no remote tracking
      if git rev-parse --verify "$BRANCH_NAME" &>/dev/null; then
        git branch -D "$BRANCH_NAME" 2>/dev/null || true
        echo "🗑️  Deleted branch: $BRANCH_NAME"
      fi
      echo "✅ Worktree and branch cleaned up"
    else
      echo "ℹ️  No worktree found at $WT_PATH — nothing to abandon"
    fi
    exit 0
  fi

  # ─── Recovery detection (R3: Failure Recovery) ──────────────────────────
  # If worktree exists with stale ralph-loop state, a previous build was
  # interrupted. Surface state and options to the user.
  if [[ -d "$WT_PATH" ]] && [[ -f "$WT_PATH/.claude/ralph-loop.local.md" ]]; then
    # Extract last commit info from worktree
    WT_LAST_COMMIT=$(cd "$WT_PATH" && git log --oneline -1 2>/dev/null || echo "unknown")
    WT_DIFF_STAT=$(cd "$WT_PATH" && git diff --stat "main...HEAD" 2>/dev/null | tail -1 || echo "no changes")
    WT_BRANCH=$(cd "$WT_PATH" && git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "$BRANCH_NAME")

    echo "⚠️  Previous build session detected in worktree" >&2
    echo "" >&2
    echo "   Branch:      $WT_BRANCH" >&2
    echo "   Last commit:  $WT_LAST_COMMIT" >&2
    echo "   Diff stats:   $WT_DIFF_STAT" >&2
    echo "" >&2
    echo "   Resuming build (merge main + verify env before restart)..." >&2
    # Clean stale ralph-loop state so a fresh loop starts
    rm -f "$WT_PATH/.claude/ralph-loop.local.md"
  fi

  if [[ -d "$WT_PATH" ]]; then
    echo "📂 Using existing worktree: $WT_PATH"

    # ─── Auto-merge main into worktree branch (R1: Auto-Merge on Resume) ───
    # Ensure the agent always works with the latest code from main.
    MERGE_BASE_BRANCH="main"
    # Fetch latest main so we merge up-to-date content
    git fetch origin "$MERGE_BASE_BRANCH" 2>/dev/null || true

    # Perform the merge inside the worktree directory
    MERGE_OUTPUT=$(cd "$WT_PATH" && git merge "origin/$MERGE_BASE_BRANCH" --no-edit 2>&1) || {
      MERGE_EXIT=$?
      if echo "$MERGE_OUTPUT" | grep -q "CONFLICT"; then
        # Abort the failed merge to leave worktree in clean state
        (cd "$WT_PATH" && git merge --abort 2>/dev/null || true)
        echo "❌ Merge conflicts detected merging $MERGE_BASE_BRANCH into $BRANCH_NAME" >&2
        echo "" >&2
        echo "Conflicting files:" >&2
        echo "$MERGE_OUTPUT" | grep "CONFLICT" >&2
        echo "" >&2
        echo "Options:" >&2
        echo "  1. Resolve manually: cd $WT_PATH && git merge origin/$MERGE_BASE_BRANCH" >&2
        echo "  2. Abort and recreate: git worktree remove $WT_PATH && re-run build" >&2
        echo "  3. Continue without merge (worktree may be stale)" >&2
        exit 1
      else
        echo "⚠️  Merge failed (exit $MERGE_EXIT): $MERGE_OUTPUT" >&2
        echo "   Continuing with existing worktree state." >&2
      fi
    }

    # Log merge result if successful
    if [[ $? -eq 0 ]] && [[ -n "$MERGE_OUTPUT" ]]; then
      if echo "$MERGE_OUTPUT" | grep -q "Already up to date"; then
        echo "✅ Worktree already up to date with $MERGE_BASE_BRANCH"
      else
        echo "✅ Merged $MERGE_BASE_BRANCH into $BRANCH_NAME"
        echo "   $MERGE_OUTPUT" | head -5
      fi
    fi
  else
    # Create branch if needed
    if ! git rev-parse --verify "$BRANCH_NAME" &>/dev/null; then
      git branch "$BRANCH_NAME" HEAD 2>/dev/null || true
    fi
    git worktree add "$WT_PATH" "$BRANCH_NAME" 2>/dev/null || {
      git worktree add --force "$WT_PATH" "$BRANCH_NAME" 2>/dev/null || true
    }
    echo "📂 Created worktree: $WT_PATH (branch: $BRANCH_NAME)"
  fi

  # ─── Forward .env* files via symlinks (R2: Env Var Forwarding) ──────────
  # Symlinks auto-reflect changes in project root without manual sync.
  ENV_FORWARDED=0
  for env_file in "$PROJECT_ROOT"/.env*; do
    [[ -f "$env_file" ]] || continue
    env_basename="$(basename "$env_file")"
    target="$WT_PATH/$env_basename"
    if [[ -L "$target" ]]; then
      # Already a symlink — verify it points to the right place
      if [[ "$(readlink "$target")" != "$env_file" ]]; then
        rm -f "$target"
        ln -s "$env_file" "$target"
        ENV_FORWARDED=$((ENV_FORWARDED + 1))
      fi
    elif [[ -f "$target" ]]; then
      # Regular file exists — replace with symlink for auto-sync
      rm -f "$target"
      ln -s "$env_file" "$target"
      ENV_FORWARDED=$((ENV_FORWARDED + 1))
    else
      ln -s "$env_file" "$target"
      ENV_FORWARDED=$((ENV_FORWARDED + 1))
    fi
  done
  if [[ $ENV_FORWARDED -gt 0 ]]; then
    echo "🔗 Forwarded $ENV_FORWARDED .env* file(s) to worktree via symlinks"
  fi

  # ─── Worktree health check (R4: Worktree Health Check) ──────────────────
  HEALTH_WARNINGS=0
  HEALTH_ERRORS=0

  # Error: worktree path doesn't exist (shouldn't happen here, but defensive)
  if [[ ! -d "$WT_PATH" ]]; then
    echo "❌ HEALTH ERROR: Worktree path does not exist: $WT_PATH" >&2
    HEALTH_ERRORS=$((HEALTH_ERRORS + 1))
  fi

  # Error: detached HEAD
  WT_HEAD=$(cd "$WT_PATH" && git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
  if [[ "$WT_HEAD" == "HEAD" ]]; then
    echo "❌ HEALTH ERROR: Worktree is in detached HEAD state" >&2
    HEALTH_ERRORS=$((HEALTH_ERRORS + 1))
  fi

  # Error: branch deleted
  if [[ -n "$WT_HEAD" ]] && [[ "$WT_HEAD" != "HEAD" ]]; then
    if ! git rev-parse --verify "$WT_HEAD" &>/dev/null 2>&1; then
      echo "❌ HEALTH ERROR: Branch '$WT_HEAD' no longer exists" >&2
      HEALTH_ERRORS=$((HEALTH_ERRORS + 1))
    fi
  fi

  # Warning: uncommitted changes
  WT_STATUS=$(cd "$WT_PATH" && git status --porcelain 2>/dev/null || echo "")
  if [[ -n "$WT_STATUS" ]]; then
    WT_DIRTY_COUNT=$(echo "$WT_STATUS" | wc -l | tr -d ' ')
    echo "⚠️  HEALTH WARNING: $WT_DIRTY_COUNT uncommitted change(s) in worktree" >&2
    HEALTH_WARNINGS=$((HEALTH_WARNINGS + 1))
  fi

  # Warning: env files missing (check if project root has env files but worktree doesn't)
  for env_file in "$PROJECT_ROOT"/.env*; do
    [[ -f "$env_file" ]] || continue
    env_basename="$(basename "$env_file")"
    if [[ ! -e "$WT_PATH/$env_basename" ]]; then
      echo "⚠️  HEALTH WARNING: $env_basename missing from worktree" >&2
      HEALTH_WARNINGS=$((HEALTH_WARNINGS + 1))
    fi
  done

  # Write health check results for TUI consumption
  mkdir -p "$WT_PATH/.claude"
  cat > "$WT_PATH/.claude/health-check.json" <<HEALTH_EOF
{"timestamp":"$(date -u +%Y-%m-%dT%H:%M:%SZ)","errors":$HEALTH_ERRORS,"warnings":$HEALTH_WARNINGS,"branch":"$WT_HEAD","status":"$(if [[ $HEALTH_ERRORS -gt 0 ]]; then echo "error"; elif [[ $HEALTH_WARNINGS -gt 0 ]]; then echo "warning"; else echo "healthy"; fi)"}
HEALTH_EOF

  if [[ $HEALTH_ERRORS -gt 0 ]]; then
    echo "❌ Health check failed with $HEALTH_ERRORS error(s). Build cannot proceed." >&2
    exit 1
  elif [[ $HEALTH_WARNINGS -gt 0 ]]; then
    echo "⚠️  Health check passed with $HEALTH_WARNINGS warning(s)"
  else
    echo "✅ Health check passed"
  fi

  # Switch to the worktree
  cd "$WT_PATH"
  echo "📂 Working in: $(pwd)"
  echo "BLUEPRINT_WORKTREE_PATH=$WT_PATH"
fi

# ─── Find frontier (smart selection) ────────────────────────────────────────
#
# Strategy:
#   1. Only search context/sites/ (not context/plans/ — false positives)
#   2. Exclude archive/ subdirectory (completed frontiers)
#   3. If --filter is set, match filter anywhere in filename
#   4. Rank candidates: in-progress worktree > has incomplete tasks > rest
#   5. If multiple ambiguous candidates, list them for visibility

FRONTIER_FILE=""
ALL_CANDIDATES=()

if [[ -d "context/sites" ]]; then
  while IFS= read -r -d '' f; do
    # Skip archive directory
    [[ "$f" == *"/archive/"* ]] && continue
    # Skip non-site files (must have "site" in name)
    [[ "$(basename "$f")" != *site* ]] && continue
    ALL_CANDIDATES+=("$f")
  done < <(find "context/sites" -maxdepth 1 -name "*.md" -type f -print0 2>/dev/null | sort -z)
fi

# Apply filter if set — match filter anywhere in filename
CANDIDATES=()
if [[ ${#ALL_CANDIDATES[@]} -gt 0 ]]; then
  if [[ -n "$FILTER" ]]; then
    for f in "${ALL_CANDIDATES[@]}"; do
      bn="$(basename "$f")"
      if [[ "$bn" == *"$FILTER"* ]]; then
        CANDIDATES+=("$f")
      fi
    done
    # If filter matched nothing, fall back to all candidates
    if [[ ${#CANDIDATES[@]} -eq 0 ]]; then
      echo "⚠️  Filter '$FILTER' matched no build sites, searching all" >&2
      CANDIDATES=("${ALL_CANDIDATES[@]}")
    fi
  else
    CANDIDATES=("${ALL_CANDIDATES[@]}")
  fi
fi

if [[ ${#CANDIDATES[@]} -eq 0 ]]; then
  echo "❌ No build site found in context/sites/" >&2
  echo "   Run /bp:architect first to generate one." >&2
  # Also check context/plans/ as a hint
  if [[ -d "context/plans" ]] && find "context/plans" -name "*site*" -type f 2>/dev/null | grep -q .; then
    echo "   (Found frontier files in context/plans/ — move them to context/sites/)" >&2
  fi
  exit 1
fi

# If exactly one candidate, use it directly
if [[ ${#CANDIDATES[@]} -eq 1 ]]; then
  FRONTIER_FILE="${CANDIDATES[0]}"
else
  # Multiple candidates — rank by status
  # Priority: has active worktree > has incomplete tasks > alphabetical
  BEST_SCORE=0
  for f in "${CANDIDATES[@]}"; do
    score=1  # base score

    # Check for active worktree (in-progress = highest priority)
    bn="$(basename "$f" .md)"
    # Derive worktree name the same way the picker does
    wt_name=$(echo "$bn" | sed -E 's/^(plan-|feature-frontier-|feature-|build-site-)//' | sed -E 's/-?frontier-?//' | sed -E 's/^-|-$//g')
    [[ -z "$wt_name" ]] && wt_name="build"
    wt_path="${PROJECT_ROOT}/../${PROJECT_NAME}-blueprint-${wt_name}"

    if [[ -d "$wt_path" ]]; then
      if [[ -f "$wt_path/.claude/ralph-loop.local.md" ]]; then
        score=3  # active worktree with ralph loop
      else
        score=2  # worktree exists (maybe stale, but likely relevant)
      fi
    fi

    # Check if frontier has incomplete tasks (prefer non-done frontiers)
    if [[ $score -lt 2 ]]; then
      task_count=$(grep -cE '\|\s*T-([A-Za-z0-9]+-)*[0-9]+\s*\|' "$f" 2>/dev/null || echo 0)
      if [[ $task_count -gt 0 ]]; then
        # Check if all tasks are done
        done_count=0
        if [[ -d "context/impl" ]]; then
          for task_id in $(grep -oE 'T-([A-Za-z0-9]+-)*[0-9]+' "$f" 2>/dev/null | sort -u); do
            if grep -rlq "$task_id.*DONE\|DONE.*$task_id" context/impl/ 2>/dev/null; then
              done_count=$((done_count + 1))
            fi
          done
        fi
        if [[ $done_count -lt $task_count ]]; then
          score=2  # has incomplete tasks
        fi
      fi
    fi

    if [[ $score -gt $BEST_SCORE ]]; then
      BEST_SCORE=$score
      FRONTIER_FILE="$f"
    fi
  done

  # List all candidates so Claude has visibility
  echo "📋 Found ${#CANDIDATES[@]} frontiers:" >&2
  for f in "${CANDIDATES[@]}"; do
    marker="  "
    [[ "$f" == "$FRONTIER_FILE" ]] && marker="→ "
    task_count=$(grep -cE '\|\s*T-([A-Za-z0-9]+-)*[0-9]+\s*\|' "$f" 2>/dev/null || echo "?")
    echo "  ${marker}$(basename "$f") (${task_count} tasks)" >&2
  done
fi

echo "📋 Frontier: $FRONTIER_FILE"

# ─── Auto-archive previous cycle ────────────────────────────────────────────

ARCHIVE_COUNT=0
if [[ -d "context/impl" ]]; then
  HAS_OLD=false
  [[ -f "context/impl/loop-log.md" ]] && HAS_OLD=true
  for f in context/impl/impl-*.md; do
    [[ -f "$f" ]] && HAS_OLD=true && break
  done

  if [[ "$HAS_OLD" == "true" ]]; then
    ARCHIVE_DIR="context/impl/archive/$(date -u +%Y%m%d-%H%M%S)"
    mkdir -p "$ARCHIVE_DIR"

    for f in context/impl/loop-log.md context/impl/peer-review-findings.md context/peer-review-findings.md; do
      [[ -f "$f" ]] && mv "$f" "$ARCHIVE_DIR/" && ARCHIVE_COUNT=$((ARCHIVE_COUNT + 1))
    done
    for f in context/impl/impl-*.md; do
      [[ -f "$f" ]] && [[ "$(basename "$f")" != "CLAUDE.md" ]] && mv "$f" "$ARCHIVE_DIR/" && ARCHIVE_COUNT=$((ARCHIVE_COUNT + 1))
    done

    if [[ $ARCHIVE_COUNT -gt 0 ]]; then
      echo "📦 Archived $ARCHIVE_COUNT files from previous cycle → $ARCHIVE_DIR/"
    fi
  fi
fi

# Remove old ralph state
rm -f .claude/ralph-loop.local.md

# ─── Discover specs and refs ────────────────────────────────────────────────

SPEC_FILES=()
if [[ -d "context/blueprints" ]]; then
  while IFS= read -r -d '' f; do
    [[ "$(basename "$f")" == "CLAUDE.md" ]] && continue
    if [[ -n "$FILTER" ]] && [[ "$f" != *"$FILTER"* ]]; then continue; fi
    SPEC_FILES+=("$f")
  done < <(find context/blueprints -name "*.md" -type f -print0 2>/dev/null | sort -z)
fi

SPEC_LISTING=""
for f in "${SPEC_FILES[@]+"${SPEC_FILES[@]}"}"; do
  SPEC_LISTING="${SPEC_LISTING}\n- \`$f\`"
done

# ─── Configure Codex MCP if peer review ─────────────────────────────────────

if [[ ""$PEER_REVIEW"" == "true" ]]; then
  MCP_FILE=".mcp.json"
  NEEDS_MCP=false

  if [[ ! -f "$MCP_FILE" ]]; then
    NEEDS_MCP=true
  elif ! python3 -c "
import json, sys
with open('$MCP_FILE') as f:
    d = json.load(f)
sys.exit(0 if 'codex-reviewer' in d.get('mcpServers', {}) else 1)
" 2>/dev/null; then
    NEEDS_MCP=true
  fi

  if [[ "$NEEDS_MCP" == "true" ]]; then
    if ! command -v codex &>/dev/null; then
      echo "❌ Codex CLI not found. Install: npm install -g @openai/codex" >&2
      exit 1
    fi
    if [[ -f "$MCP_FILE" ]]; then
      python3 -c "
import json
with open('$MCP_FILE') as f:
    d = json.load(f)
d.setdefault('mcpServers', {})['codex-reviewer'] = {
    'command': 'codex',
    'args': ['mcp-server', '-c', 'model=\"$CODEX_MODEL\"']
}
with open('$MCP_FILE', 'w') as f:
    json.dump(d, f, indent=2)
"
    else
      python3 -c "
import json
d = {'mcpServers': {'codex-reviewer': {'command': 'codex', 'args': ['mcp-server', '-c', 'model=\"$CODEX_MODEL\"']}}}
with open('$MCP_FILE', 'w') as f:
    json.dump(d, f, indent=2)
"
    fi
    echo "📡 Configured Codex ($CODEX_MODEL) as MCP peer reviewer"
  fi
fi

# ─── Build prompt ───────────────────────────────────────────────────────────

PEER_REVIEW_SECTION=""
if [[ ""$PEER_REVIEW"" == "true" ]]; then
  PEER_REVIEW_SECTION="
## Peer Review (every ${REVIEW_INTERVAL}th iteration)

Check the iteration number from the Ralph system message.
If iteration % $REVIEW_INTERVAL == 0, this is a REVIEW iteration:

1. Run \`git diff main...HEAD\` to get all changes
2. Call the \`codex-reviewer\` MCP server with this prompt:

   > You are a senior engineer performing peer review code review.
   > Your job is to find what the builder MISSED — not to agree.
   > SPEC REQUIREMENTS: [include relevant spec content]
   > CODE CHANGES: [include diff]
   > For each issue: Severity (CRITICAL/HIGH/MEDIUM/LOW), File, Issue, Suggestion.
   > If you find zero issues, explain what you checked and why it's correct.

3. Write findings to \`context/impl/peer-review-findings.md\`
4. Fix all CRITICAL and HIGH findings immediately
5. Mark fixed findings as FIXED

Completion requires: no CRITICAL/HIGH findings remain unfixed."
fi

RALPH_PROMPT="# Blueprint Build

## Your Role
You are implementing tasks from a build site. Each iteration: find the next
unblocked task, read its blueprint, implement it, validate, commit.

## Read These First (every iteration)
1. \`context/impl/loop-log.md\` — your iteration history (if exists)
2. \`$FRONTIER_FILE\` — the task dependency graph
3. Any \`context/impl/impl-*.md\` files — per-domain progress

## Blueprints (read when implementing a specific requirement)
$(echo -e "$SPEC_LISTING")
"$PEER_REVIEW"_SECTION
## Each Iteration

### 1. Orient
- Read loop-log.md and impl tracking to know what's done
- Read the build site to find the lowest tier with incomplete tasks

### 2. Pick Task
- Find the next unblocked task (all blockedBy tasks are DONE)
- Among equals, pick the one that unblocks the most downstream work

### 3. Implement
- Read the task's blueprint requirement and acceptance criteria
- Implement it, following existing codebase patterns
- One task per iteration

### 4. Validate
1. **Build** — must compile/pass
2. **Tests** — on changed files, must pass
3. **Acceptance criteria** — each criterion from the spec must be met

If stuck 2+ attempts → document as dead end, move on.

### 5. Track
Update \`context/impl/impl-{domain}.md\` (create if missing):

\`\`\`markdown
---
created: \"{CURRENT_DATE_UTC}\"
last_edited: \"{CURRENT_DATE_UTC}\"
---
# Implementation Tracking: {domain}
| Task | Status | Notes |
|------|--------|-------|
| T-001 | DONE | what was done |
\`\`\`

Append to \`context/impl/loop-log.md\` (create if missing):

\`\`\`markdown
### Iteration N — {timestamp}
- **Task:** T-{id} — {title}
- **Tier:** {n}
- **Status:** DONE / PARTIAL / BLOCKED
- **Files:** {changed files}
- **Validation:** Build {P/F}, Tests {P/F}, Acceptance {n/n}
- **Next:** T-{id} — {next task}
\`\`\`

### 6. Commit
Descriptive message with task ID and blueprint requirement. Do NOT push.

### 7. Done?
All tasks across all tiers DONE + build passes + tests pass?
→ output: <promise>$COMPLETION_PROMISE</promise>

Otherwise → next iteration.

## Rules
1. NEVER output completion promise unless ALL tasks are genuinely DONE
2. ONE task per iteration
3. Stuck 2+ iterations → dead end, move on
4. Re-read build site and tracking every iteration
5. Commit after each task"

# ─── Write Ralph Loop state ─────────────────────────────────────────────────

mkdir -p .claude

cat > .claude/ralph-loop.local.md <<EOF
---
active: true
iteration: 1
session_id: ${CLAUDE_CODE_SESSION_ID:-$(python3 -c "import uuid; print(uuid.uuid4())" 2>/dev/null || echo "blueprint-$$-$(date +%s)")}
max_iterations: $MAX_ITERATIONS
completion_promise: "$COMPLETION_PROMISE"
started_at: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
---

$RALPH_PROMPT
EOF

# ─── Output ─────────────────────────────────────────────────────────────────

cat <<EOF
🔄 Blueprint Build — Loop activated!

Frontier: $FRONTIER_FILE
Specs: ${#SPEC_FILES[@]} found
$(if [[ -n "$FILTER" ]]; then echo "Filter: $FILTER"; fi)
$(if [[ ""$PEER_REVIEW"" == "true" ]]; then echo "Peer reviewer: Codex ($CODEX_MODEL) every ${REVIEW_INTERVAL} iterations"; fi)
$(if [[ $ARCHIVE_COUNT -gt 0 ]]; then echo "Archived: $ARCHIVE_COUNT files from previous cycle"; fi)
Max iterations: $MAX_ITERATIONS

Each iteration: build site → blueprint → implement → validate → commit

═══════════════════════════════════════════════════════════════════════
COMPLETION: <promise>$COMPLETION_PROMISE</promise>
Only when ALL tasks are done.
═══════════════════════════════════════════════════════════════════════

$RALPH_PROMPT
EOF
