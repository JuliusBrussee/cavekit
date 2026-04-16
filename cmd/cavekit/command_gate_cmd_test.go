package main

import "testing"

func TestFastClassify_BlockAndAllow(t *testing.T) {
	cfg := newCavekitConfig(t.TempDir())

	blocked, err := fastClassify("rm -rf /", cfg)
	if err != nil {
		t.Fatalf("fastClassify(block): %v", err)
	}
	if !stringsHasPrefix(blocked, "BLOCK|") {
		t.Fatalf("fastClassify(block) = %q", blocked)
	}

	allowed, err := fastClassify("git status", cfg)
	if err != nil {
		t.Fatalf("fastClassify(allow): %v", err)
	}
	if allowed != "APPROVE" {
		t.Fatalf("fastClassify(allow) = %q, want APPROVE", allowed)
	}
}

func TestNormalizeGateCommand(t *testing.T) {
	got := normalizeGateCommand(`git show 1234567 "./foo/bar.txt"`)
	if !stringsContains(got, "<HASH>") || !stringsContains(got, "<STR>") {
		t.Fatalf("normalizeGateCommand = %q", got)
	}
}

func TestCodexClassify_PassthroughWithoutCodex(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	cfg := newCavekitConfig(t.TempDir())
	result, err := codexClassify("custom-dangerous-command", t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("codexClassify: %v", err)
	}
	if result != "PASSTHROUGH|Codex unavailable" {
		t.Fatalf("codexClassify = %q", result)
	}
}
