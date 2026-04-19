"use strict";

const assert = require("assert");
const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname, "..");
const opencodeDir = path.join(root, "opencode");
const commandsDir = path.join(opencodeDir, "commands");
const installPath = path.join(root, "install.sh");

const expectedCommands = [
  "ck-help.md",
  "ck-init.md",
  "ck-sketch.md",
  "ck-map.md",
  "ck-make.md",
  "ck-check.md",
  "ck-status.md",
];

const bannedPatterns = [
  /CLAUDE_PLUGIN_ROOT/,
  /Agent\s*\(/,
  /subagent_type/,
  /cavekit-tools\.cjs/,
  /cavekit-router\.cjs/,
  /stop-hook/i,
  /make-parallel/,
];

function read(filePath) {
  return fs.readFileSync(filePath, "utf8");
}

module.exports = {
  "opencode port files exist"() {
    assert.ok(fs.existsSync(opencodeDir), "missing opencode/");
    assert.ok(fs.existsSync(commandsDir), "missing opencode/commands/");
    for (const file of expectedCommands) {
      assert.ok(fs.existsSync(path.join(commandsDir, file)), `missing ${file}`);
    }
    assert.ok(fs.existsSync(path.join(opencodeDir, "README.md")), "missing opencode/README.md");
    assert.ok(fs.existsSync(path.join(opencodeDir, "AGENTS.md")), "missing opencode/AGENTS.md");
  },

  "opencode commands do not leak upstream runtime tokens"() {
    for (const file of expectedCommands) {
      const body = read(path.join(commandsDir, file));
      for (const pattern of bannedPatterns) {
        assert.ok(!pattern.test(body), `${file} leaked banned pattern ${pattern}`);
      }
      assert.ok(/OpenCode Cavekit portable port|OpenCode Cavekit portable workflow/.test(body), `${file} missing portability scope marker`);
    }
  },

  "install script wires opencode commands"() {
    const install = read(installPath);
    assert.match(install, /OPENCODE_COMMANDS_DIR/, "install.sh missing OpenCode commands dir variable");
    assert.match(install, /LOCAL_BIN_DIR/, "install.sh missing local bin fallback variable");
    assert.match(install, /\.local\/bin/, "install.sh missing local bin fallback path");
    assert.match(install, /opencode\/commands\//, "install.sh missing opencode commands sync");
    assert.match(install, /Configuring OpenCode commands/, "install.sh missing OpenCode sync step");
    assert.match(install, /ck-help/, "install.sh missing OpenCode command summary");
  },

  "opencode readme states honest limitations"() {
    const readme = read(path.join(opencodeDir, "README.md"));
    assert.match(readme, /Portable Phase 1/, "README missing phase label");
    assert.match(readme, /not the Claude Code plugin runtime/i, "README missing non-parity warning");
    assert.match(readme, /No autonomous stop-hook loop/i, "README missing runtime limitation");
    assert.match(readme, /\/ck-make/i, "README missing command docs");
  },
};
