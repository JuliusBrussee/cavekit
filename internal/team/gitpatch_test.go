package team

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureGitignoreBlock_Idempotent(t *testing.T) {
	root := t.TempDir()
	gitignore := filepath.Join(root, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	if err := EnsureGitignoreBlock(root, false); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	if err := EnsureGitignoreBlock(root, false); err != nil {
		t.Fatalf("second ensure: %v", err)
	}

	data, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	text := string(data)
	if strings.Count(text, managedBlockStart) != 1 {
		t.Fatalf("expected one managed block, got:\n%s", text)
	}
	if !strings.Contains(text, ".cavekit/team/leases/") || !strings.Contains(text, ".cavekit/team/identity.json") {
		t.Fatalf("managed block missing team entries:\n%s", text)
	}
}

func TestEnsureGitattributesBlock_WarnsOnConflict(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".gitattributes")
	content := ".cavekit/team/ledger.jsonl merge=ours\n*.png binary\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write .gitattributes: %v", err)
	}

	warnings, err := EnsureGitattributesBlock(root, false)
	if err != nil {
		t.Fatalf("ensure .gitattributes: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %d (%v)", len(warnings), warnings)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .gitattributes: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, ".cavekit/team/ledger.jsonl merge=ours") {
		t.Fatalf("expected original conflicting entry to remain:\n%s", text)
	}
	if !strings.Contains(text, "refs/heads/cavekit/team") {
		t.Fatalf("expected managed block to reference the ledger ref:\n%s", text)
	}
}
