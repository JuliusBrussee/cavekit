#!/bin/bash

# cavekit-launch-session — Creates a tmux session with one pane per build site.
# Each pane runs Claude in the project directory with /ck:make.
#
# Usage: cavekit-launch-session.sh [--expanded] <frontier-path> [<frontier-path> ...]
#
# Default: all panes in one window (horizontal for 2-3, tiled for 4+)
# --expanded: one window per frontier with progress+activity dashboard panes

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SESSION_NAME="cavekit"
EXPANDED=false
STAGGER_DELAY=5

# ─── Parse args ───────────────────────────────────────────────────────────────

if [[ "${1:-}" == "--expanded" ]]; then
  EXPANDED=true
  shift
fi

FRONTIERS=("$@")

if [[ ${#FRONTIERS[@]} -eq 0 ]]; then
  echo "Usage: cavekit-launch-session.sh [--expanded] <frontier-path> ..." >&2
  exit 1
fi

# ─── Preflight ────────────────────────────────────────────────────────────────

command -v tmux &>/dev/null || { echo "tmux not found. Install: brew install tmux" >&2; exit 1; }
command -v claude &>/dev/null || { echo "claude not found." >&2; exit 1; }

PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

# Kill existing session if running
if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
  echo "Existing '$SESSION_NAME' session found. Killing it..."
  tmux kill-session -t "$SESSION_NAME"
fi

# ─── Derive frontier names ───────────────────────────────────────────────────

derive_name() {
  basename "$1" .md | sed -E 's/^(plan-|feature-frontier-|feature-|build-site-)//' | sed 's/-frontier$//'
}

# ─── Derive names and check for resumable sessions ───────────────────────────

NAMES=()
RESUMING=()  # "true" or "false" per frontier

for frontier in "${FRONTIERS[@]}"; do
  name=$(derive_name "$frontier")
  NAMES+=("$name")

  # Check if this is a resumable session (has ralph-loop state or impl progress)
  if [[ -f "$PROJECT_ROOT/.claude/ralph-loop.local.md" ]] || \
     ls "$PROJECT_ROOT/context/impl/impl-"*.md &>/dev/null 2>&1; then
    RESUMING+=("true")
    echo "  $name: will resume existing session"
  else
    RESUMING+=("false")
  fi
done

# ─── Write launcher script for each pane ─────────────────────────────────────

write_launcher() {
  local project_dir="$1"
  local frontier_path="$2"
  local name="$3"
  local resuming="$4"
  local launcher
  launcher=$(mktemp /tmp/cavekit-launch-${name}-XXXXXX.sh)

  local frontier_basename
  frontier_basename=$(basename "$frontier_path")

  local claude_cmd="claude"
  local mode_label="NEW"
  if [[ "$resuming" == "true" ]]; then
    claude_cmd="claude --resume"
    mode_label="RESUME"
  fi

  cat > "$launcher" <<LAUNCHER_EOF
#!/bin/bash
rm -f "$launcher"
cd "$project_dir"
echo "Cavekit Agent: $name [$mode_label]"
echo "Directory: $project_dir"
echo "Frontier: $frontier_basename"
echo ""
$claude_cmd
LAUNCHER_EOF
  chmod +x "$launcher"
  echo "$launcher"
}

# ─── Build tmux session ─────────────────────────────────────────────────────

if [[ "$EXPANDED" == "true" ]]; then
  # ── Expanded mode: one window per frontier with 3-pane layout ──

  FIRST=true
  WIN_IDX=0

  for i in "${!FRONTIERS[@]}"; do
    name="${NAMES[$i]}"
    frontier="${FRONTIERS[$i]}"
    launcher=$(write_launcher "$PROJECT_ROOT" "$frontier" "$name" "${RESUMING[$i]}")

    if [[ "$FIRST" == "true" ]]; then
      # Create session with first window
      tmux new-session -d -s "$SESSION_NAME" -n "$name" -c "$PROJECT_ROOT" \
        "bash $launcher; exec bash"
      FIRST=false
    else
      # Add new window
      tmux new-window -t "$SESSION_NAME" -n "$name" -c "$PROJECT_ROOT" \
        "bash $launcher; exec bash"
    fi

    # Add dashboard panes (right side)
    RIGHT_WIDTH=$(( $(tmux display-message -t "$SESSION_NAME:${WIN_IDX}" -p '#{window_width}') * 30 / 100 ))
    [[ "$RIGHT_WIDTH" -lt 35 ]] && RIGHT_WIDTH=35

    tmux split-window -h -t "$SESSION_NAME:${WIN_IDX}" -l "$RIGHT_WIDTH" -c "$PROJECT_ROOT" \
      "exec bash \"$SCRIPT_DIR/dashboard-progress.sh\""
    tmux select-pane -T "cavekit-progress"

    tmux split-window -v -t "$SESSION_NAME:${WIN_IDX}" -c "$PROJECT_ROOT" \
      "exec bash \"$SCRIPT_DIR/dashboard-activity.sh\""
    tmux select-pane -T "cavekit-activity"

    # Focus main pane
    tmux select-pane -t "$SESSION_NAME:${WIN_IDX}.0"

    WIN_IDX=$((WIN_IDX + 1))
  done

  # Select first window
  tmux select-window -t "$SESSION_NAME:0"

else
  # ── Default mode: all panes in one window ──

  FIRST=true
  PANE_IDX=0

  for i in "${!FRONTIERS[@]}"; do
    name="${NAMES[$i]}"
    frontier="${FRONTIERS[$i]}"
    launcher=$(write_launcher "$PROJECT_ROOT" "$frontier" "$name" "${RESUMING[$i]}")

    if [[ "$FIRST" == "true" ]]; then
      tmux new-session -d -s "$SESSION_NAME" -n "cavekit-agents" -c "$PROJECT_ROOT" \
        "bash $launcher; exec bash"
      FIRST=false
    else
      # Split from the first pane to add more
      tmux split-window -t "$SESSION_NAME:0" -c "$PROJECT_ROOT" \
        "bash $launcher; exec bash"
    fi

    PANE_IDX=$((PANE_IDX + 1))
  done

  # Apply layout based on pane count
  if [[ $PANE_IDX -le 3 ]]; then
    tmux select-layout -t "$SESSION_NAME:0" even-horizontal
  else
    tmux select-layout -t "$SESSION_NAME:0" tiled
  fi

  # Select first pane
  tmux select-pane -t "$SESSION_NAME:0.0"
fi

# ─── Staggered /ck:make launch ──────────────────────────────────────

# Background process that sends /ck:make to NEW panes (resumed ones already have context)
(
  sleep 3  # Wait for Claude instances to start

  for i in "${!NAMES[@]}"; do
    name="${NAMES[$i]}"

    # Resumed sessions already have their loop — skip sending /ck:make
    if [[ "${RESUMING[$i]}" == "true" ]]; then
      continue
    fi

    if [[ "$EXPANDED" == "true" ]]; then
      target="$SESSION_NAME:${i}.0"
    else
      target="$SESSION_NAME:0.${i}"
    fi

    tmux send-keys -t "$target" "/ck:make --filter ${name}" Enter

    # Stagger between launches (skip delay after last one)
    sleep "$STAGGER_DELAY"
  done
) &

# ─── Start status poller ────────────────────────────────────────────────────

if [[ -x "$SCRIPT_DIR/cavekit-status-poller.sh" ]]; then
  "$SCRIPT_DIR/cavekit-status-poller.sh" &
fi

# ─── Enable mouse mode ───────────────────────────────────────────────────────

tmux set-option -t "$SESSION_NAME" mouse on 2>/dev/null || true

# ─── Report & attach ────────────────────────────────────────────────────────

echo ""
echo "Launched ${#FRONTIERS[@]} Cavekit agents in $PROJECT_ROOT:"
for i in "${!NAMES[@]}"; do
  echo "  ${NAMES[$i]}"
done
echo ""
echo "Attaching to tmux session '$SESSION_NAME'..."
echo "  Switch panes: Ctrl-b + arrow keys"
if [[ "$EXPANDED" == "true" ]]; then
  echo "  Switch windows: Ctrl-b + number"
fi
echo "  Detach: Ctrl-b d"
echo "  Kill all: cavekit --kill"
echo ""

exec tmux attach-session -t "$SESSION_NAME"
