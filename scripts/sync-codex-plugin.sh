#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGIN_NAME="ck"
PLUGIN_DIR="$HOME/plugins/$PLUGIN_NAME"
MARKETPLACE_FILE="$HOME/.agents/plugins/marketplace.json"
LEGACY_LINK="$HOME/.codex/cavekit"
PROMPTS_DIR="$HOME/.codex/prompts"

R=$'\033[0m' B=$'\033[1m' GR=$'\033[32m' YL=$'\033[33m' BL=$'\033[34m' RD=$'\033[31m'

info()  { printf "${BL}▸${R} %s\n" "$1"; }
ok()    { printf "${GR}■${R} %s\n" "$1"; }
warn()  { printf "${YL}!${R} %s\n" "$1"; }
fail()  { printf "${RD}✗${R} %s\n" "$1" >&2; exit 1; }

command -v python3 &>/dev/null || fail "python3 not found."

info "Syncing Cavekit into Codex local plugins..."

mkdir -p "$HOME/plugins" "$(dirname "$MARKETPLACE_FILE")" "$PROMPTS_DIR"
ln -sfn "$ROOT_DIR" "$PLUGIN_DIR"
ok "Linked plugin at $PLUGIN_DIR"

if [[ -d "$HOME/.codex" ]]; then
  ln -sfn "$ROOT_DIR" "$LEGACY_LINK"
  ok "Updated legacy Codex shortcut at $LEGACY_LINK"
else
  warn "Skipping legacy ~/.codex symlink because ~/.codex does not exist"
fi

for command_file in "$ROOT_DIR"/commands/*.md; do
  command_name="$(basename "$command_file" .md)"
  # Primary ck- prefix
  ln -sfn "$command_file" "$PROMPTS_DIR/ck-$command_name.md"
  # Deprecated bp- alias
  ln -sfn "$command_file" "$PROMPTS_DIR/bp-$command_name.md"
done

# Clean up stale prompts for both prefixes
for prefix in ck bp; do
  for prompt_path in "$PROMPTS_DIR"/${prefix}-*.md; do
    [[ -e "$prompt_path" || -L "$prompt_path" ]] || continue
    prompt_name="$(basename "$prompt_path")"
    command_name="${prompt_name#${prefix}-}"
    command_name="${command_name%.md}"
    if [[ ! -f "$ROOT_DIR/commands/$command_name.md" ]]; then
      rm -f "$prompt_path"
    fi
  done
done
ok "Linked Codex prompts at $PROMPTS_DIR"

python3 - "$MARKETPLACE_FILE" <<'PYEOF'
import json
import os
import sys

path = sys.argv[1]
entry = {
    "name": "ck",
    "source": {
        "source": "local",
        "path": "./plugins/ck",
    },
    "policy": {
        "installation": "AVAILABLE",
        "authentication": "ON_INSTALL",
    },
    "category": "Productivity",
}

if os.path.exists(path):
    with open(path) as f:
        data = json.load(f)
else:
    data = {
        "name": "local-plugins",
        "interface": {
            "displayName": "Local Plugins",
        },
        "plugins": [],
    }

plugins = data.setdefault("plugins", [])
existing_index = next((i for i, plugin in enumerate(plugins) if plugin.get("name") in ("ck", "bp")), None)
if existing_index is None:
    plugins.append(entry)
else:
    plugins[existing_index] = entry

data.setdefault("name", "local-plugins")
data.setdefault("interface", {})
data["interface"].setdefault("displayName", "Local Plugins")

with open(path, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
PYEOF

ok "Updated Codex marketplace at $MARKETPLACE_FILE"
printf "\n${B}${GR}Codex sync complete.${R}\n"
printf "  Restart Codex if it is already running.\n"
