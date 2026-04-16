package main

import (
	"path/filepath"
	"testing"
)

func TestCavekitConfig_PrecedenceAndModels(t *testing.T) {
	homeDir := t.TempDir()
	projectRoot := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("BP_PROJECT_ROOT", projectRoot)

	cfg := newCavekitConfig(projectRoot)
	if err := cfg.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if err := cfg.Set(scopeGlobal, "bp_model_preset", "balanced"); err != nil {
		t.Fatalf("Set global: %v", err)
	}
	if err := cfg.Set(scopeProject, "bp_model_preset", "fast"); err != nil {
		t.Fatalf("Set project: %v", err)
	}

	value, err := cfg.Get("bp_model_preset", "")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if value != "fast" {
		t.Fatalf("Get = %q, want %q", value, "fast")
	}

	source, err := cfg.GetSource("bp_model_preset")
	if err != nil {
		t.Fatalf("GetSource: %v", err)
	}
	if source != "project" {
		t.Fatalf("GetSource = %q, want project", source)
	}

	model, err := cfg.Model("execution")
	if err != nil {
		t.Fatalf("Model: %v", err)
	}
	if model != "sonnet" {
		t.Fatalf("Model(execution) = %q, want sonnet", model)
	}

	show, err := cfg.Show()
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if !stringsContains(show, "bp_model_preset=fast") {
		t.Fatalf("Show missing effective preset: %s", show)
	}
	if !stringsContains(show, filepath.Join(projectRoot, ".cavekit", "config")) {
		t.Fatalf("Show missing project config path: %s", show)
	}
}
