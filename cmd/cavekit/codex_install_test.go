package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncCodexPlugin_CopiesPromptsAndMarketplace(t *testing.T) {
	sourceRoot := t.TempDir()
	homeDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(sourceRoot, "commands"), 0o755); err != nil {
		t.Fatalf("mkdir commands: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "commands", "judge.md"), []byte("# judge\n"), 0o644); err != nil {
		t.Fatalf("write command: %v", err)
	}

	if err := syncCodexPlugin(sourceRoot, homeDir); err != nil {
		t.Fatalf("syncCodexPlugin: %v", err)
	}

	for _, rel := range []string{
		filepath.Join("plugins", "ck"),
		filepath.Join(".codex", "prompts", "ck-judge.md"),
		filepath.Join(".codex", "prompts", "bp-judge.md"),
		filepath.Join(".agents", "plugins", "marketplace.json"),
	} {
		if _, err := os.Stat(filepath.Join(homeDir, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
}

func TestConfigureClaudeAndInstallBinary(t *testing.T) {
	sourceRoot := t.TempDir()
	homeDir := t.TempDir()
	binDir := t.TempDir()
	selfPath := filepath.Join(t.TempDir(), binaryFileName())

	if err := os.WriteFile(selfPath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("write self: %v", err)
	}

	t.Setenv("HOME", homeDir)
	t.Setenv("CAVEKIT_SELF_PATH", selfPath)
	t.Setenv("CAVEKIT_INSTALL_BIN_DIR", binDir)

	target, err := installBinaryPath()
	if err != nil {
		t.Fatalf("installBinaryPath: %v", err)
	}
	if err := installBinary(target); err != nil {
		t.Fatalf("installBinary: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("installed binary missing: %v", err)
	}

	if err := configureClaude(sourceRoot, homeDir); err != nil {
		t.Fatalf("configureClaude: %v", err)
	}

	for _, rel := range []string{
		filepath.Join(".claude", "settings.json"),
		filepath.Join(".claude", "plugins", "local", "cavekit-marketplace", ".claude-plugin", "marketplace.json"),
		filepath.Join(".claude", "plugins", "local", "cavekit-marketplace", ".claude-plugin", "plugin.json"),
	} {
		if _, err := os.Stat(filepath.Join(homeDir, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
}

func TestConfigureClaude_PreservesGitHubMarketplaceRepo(t *testing.T) {
	sourceRoot := t.TempDir()
	homeDir := t.TempDir()
	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("mkdir settings dir: %v", err)
	}

	initial := `{
  "extraKnownMarketplaces": {
    "impeccable": {
      "source": {
        "source": "github",
        "repo": "pbakaus/impeccable"
      }
    }
  },
  "enabledPlugins": {
    "impeccable@impeccable": true
  }
}`
	if err := os.WriteFile(settingsPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	if err := configureClaude(sourceRoot, homeDir); err != nil {
		t.Fatalf("configureClaude: %v", err)
	}

	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal settings: %v", err)
	}

	marketplaces := decoded["extraKnownMarketplaces"].(map[string]any)
	impeccable := marketplaces["impeccable"].(map[string]any)
	source := impeccable["source"].(map[string]any)
	if source["repo"] != "pbakaus/impeccable" {
		t.Fatalf("repo = %v, want %q", source["repo"], "pbakaus/impeccable")
	}
	if _, found := source["path"]; found {
		t.Fatalf("expected github marketplace source to omit path, got %v", source["path"])
	}
}

func TestRunCodexReview_GracefullySkipsWithoutCodex(t *testing.T) {
	projectRoot := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("BP_PROJECT_ROOT", projectRoot)

	output := captureStdout(t, func() {
		runCodexReview(nil)
	})
	if !stringsContains(output, "Codex is not available") {
		t.Fatalf("unexpected output: %s", output)
	}
}
