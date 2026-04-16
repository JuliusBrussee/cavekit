package main

import (
	"os"
	osexec "os/exec"
	"testing"
)

func TestRunStatus_NoWorktreesInGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := osexec.Command("git", "init", "-q")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, string(output))
	}

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("restore wd: %v", err)
		}
	}()

	output := captureStdout(t, runStatus)
	if output != "No Cavekit worktrees found.\n" {
		t.Fatalf("unexpected status output: %q", output)
	}
}
