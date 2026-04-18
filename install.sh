#!/bin/bash

# Cavekit Installer
#
# Usage:
#   git clone https://github.com/JuliusBrussee/cavekit.git ~/.cavekit && ~/.cavekit/install.sh

set -euo pipefail

INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLAUDE_DIR="$HOME/.claude"
SETTINGS_FILE="$CLAUDE_DIR/settings.json"
BIN_DIR="/usr/local/bin"
MARKETPLACE_NAME="cavekit-local"
MARKETPLACE_DIR="$CLAUDE_DIR/plugins/local/cavekit-marketplace"

# ─── Colors ─────────────────────────────────────────────────────────────────

R=$'\033[0m' B=$'\033[1m' GR=$'\033[32m' YL=$'\033[33m' BL=$'\033[34m' RD=$'\033[31m'

info()  { printf "${BL}▸${R} %s\n" "$1"; }
ok()    { printf "${GR}■${R} %s\n" "$1"; }
warn()  { printf "${YL}!${R} %s\n" "$1"; }
fail()  { printf "${RD}✗${R} %s\n" "$1" >&2; exit 1; }

# ─── Header ─────────────────────────────────────────────────────────────────

printf "\n${B}${BL}  ┌──────────────────────────┐${R}\n"
printf "${B}${BL}  │  C A V E K I T       │${R}\n"
printf "${B}${BL}  └──────────────────────────┘${R}\n"
printf "${B}Installer${R}\n\n"

# ─── Preflight ──────────────────────────────────────────────────────────────

command -v git &>/dev/null || fail "git not found."
command -v claude &>/dev/null || warn "claude CLI not found. Install Claude Code to use /ck:... commands."
command -v codex &>/dev/null || warn "codex CLI not found. Codex local plugin sync will still be configured."
command -v tmux &>/dev/null || warn "tmux not found. Install for the parallel launcher: brew install tmux"

# ─── Create marketplace with symlink to repo ─────────────────────────────

info "Setting up Cavekit marketplace..."

mkdir -p "$MARKETPLACE_DIR/.claude-plugin"

# Symlink the repo as the "ck" plugin (primary) and "bp" (deprecated alias)
ln -sfn "$INSTALL_DIR" "$MARKETPLACE_DIR/ck"
ln -sfn "$INSTALL_DIR" "$MARKETPLACE_DIR/bp"

# Write marketplace metadata
cat > "$MARKETPLACE_DIR/.claude-plugin/marketplace.json" <<EOF
{
  "name": "$MARKETPLACE_NAME",
  "owner": { "name": "$(whoami)" },
  "metadata": {
    "description": "Local Cavekit plugin marketplace",
    "version": "3.0.0"
  },
  "plugins": [
    {
      "name": "ck",
      "description": "Cavekit framework with skills, commands, agents, and references",
      "version": "3.0.0",
      "source": "./ck",
      "author": { "name": "$(whoami)" }
    },
    {
      "name": "bp",
      "description": "[DEPRECATED — use /ck:* instead] Cavekit framework (legacy alias)",
      "version": "3.0.0",
      "source": "./bp",
      "author": { "name": "$(whoami)" }
    }
  ]
}
EOF

cat > "$MARKETPLACE_DIR/.claude-plugin/plugin.json" <<EOF
{
  "name": "cavekit-marketplace",
  "description": "Local Cavekit plugin marketplace",
  "version": "3.0.0",
  "plugins": ["ck", "bp"]
}
EOF

ok "Marketplace created at $MARKETPLACE_DIR"

# ─── Register Claude Code plugin ───────────────────────────────────────────

info "Configuring Claude Code settings..."

mkdir -p "$CLAUDE_DIR"

if [[ ! -f "$SETTINGS_FILE" ]]; then
  cat > "$SETTINGS_FILE" <<EOF
{
  "extraKnownMarketplaces": {
    "${MARKETPLACE_NAME}": {
      "source": { "source": "directory", "path": "${MARKETPLACE_DIR}" }
    }
  },
  "enabledPlugins": {
    "ck@${MARKETPLACE_NAME}": true,
    "bp@${MARKETPLACE_NAME}": true
  }
}
EOF
  ok "Created $SETTINGS_FILE"
else
  if grep -q "ck@${MARKETPLACE_NAME}" "$SETTINGS_FILE" 2>/dev/null && grep -q "bp@${MARKETPLACE_NAME}" "$SETTINGS_FILE" 2>/dev/null; then
    ok "Plugin already registered"
  else
    if command -v python3 &>/dev/null; then
      python3 - "$SETTINGS_FILE" "$MARKETPLACE_NAME" "$MARKETPLACE_DIR" <<'PYEOF'
import json, sys
path, name, mpath = sys.argv[1], sys.argv[2], sys.argv[3]
with open(path) as f:
    d = json.load(f)
d.setdefault("extraKnownMarketplaces", {})[name] = {"source": {"source": "directory", "path": mpath}}
d.setdefault("enabledPlugins", {})[f"ck@{name}"] = True
d.setdefault("enabledPlugins", {})[f"bp@{name}"] = True
with open(path, "w") as f:
    json.dump(d, f, indent=2)
PYEOF
      ok "Updated $SETTINGS_FILE"
    else
      warn "Could not auto-update settings. Add manually to $SETTINGS_FILE:"
      printf "\n"
      printf '  "extraKnownMarketplaces": { "%s": { "source": { "source": "directory", "path": "%s" } } }\n' "$MARKETPLACE_NAME" "$MARKETPLACE_DIR"
      printf '  "enabledPlugins": { "ck@%s": true, "bp@%s": true }\n\n' "$MARKETPLACE_NAME" "$MARKETPLACE_NAME"
    fi
  fi
fi

# ─── Sync Codex local plugin ────────────────────────────────────────────────

info "Configuring Codex local plugin..."

chmod +x "$INSTALL_DIR/scripts/sync-codex-plugin.sh"
"$INSTALL_DIR/scripts/sync-codex-plugin.sh"

# ─── Install cavekit CLI ─────────────────────────────────────────────────

info "Installing cavekit command..."

chmod +x "$INSTALL_DIR/scripts/cavekit"
chmod +x "$INSTALL_DIR/scripts/cavekit-launch-session.sh"
chmod +x "$INSTALL_DIR/scripts/cavekit-status-poller.sh"
chmod +x "$INSTALL_DIR/scripts/cavekit-analytics.sh"
chmod +x "$INSTALL_DIR/scripts/dashboard-progress.sh"
chmod +x "$INSTALL_DIR/scripts/dashboard-activity.sh"
chmod +x "$INSTALL_DIR/scripts/setup-build.sh"

if [[ -w "$BIN_DIR" ]]; then
  ln -sf "$INSTALL_DIR/scripts/cavekit" "$BIN_DIR/cavekit"
  ok "Installed cavekit to $BIN_DIR/cavekit"
else
  info "Need sudo to install cavekit to $BIN_DIR"
  sudo ln -sf "$INSTALL_DIR/scripts/cavekit" "$BIN_DIR/cavekit"
  ok "Installed cavekit to $BIN_DIR/cavekit"
fi

# ─── Done ───────────────────────────────────────────────────────────────────

printf "\n${B}${GR}Installed!${R}\n\n"

printf "  ${B}Terminal:${R}\n"
printf "    cavekit --monitor                 Pick build sites and launch agents\n"
printf "    cavekit --monitor --expanded      One tmux window per build site\n"
printf "    cavekit --status                  Show build site progress\n"
printf "    cavekit --analytics               Show loop trends\n"
printf "    cavekit --kill                    Stop sessions\n"
printf "\n"
printf "  ${B}Claude:${R}\n"
printf "    /ck:sketch                    Draft kits\n"
printf "    /ck:map                       Generate build site\n"
printf "    /ck:make                      Build from kits\n"
printf "    /ck:check                     Inspect the build\n"
printf "    /ck:review                    Audit branch end-to-end\n"
printf "    /ck:status                    Check build progress\n"
printf "    /ck:ship <feature>            One-shot sketch → map → make → check\n"
printf "\n"
printf "  ${B}Codex:${R}\n"
printf "    Synced local plugin via ~/plugins/ck and ~/.agents/plugins/marketplace.json\n"
printf "    Linked prompts into ~/.codex/prompts (for /prompts:ck-... commands)\n"
printf "\n"
printf "  Restart Claude Code and Codex to load the plugin changes.\n\n"
