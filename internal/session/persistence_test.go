package session

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStore_SaveLoad(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "state.json")
	store := NewStore(path)

	instances := []*Instance{
		{
			Title:        "auth",
			SitePath:     "/path/site.md",
			WorktreePath: "/code/project-cavekit-auth",
			SessionName:  "auth",
			Status:       StatusRunning,
			Program:      "claude",
			CreatedAt:    time.Now(),
			TasksDone:    3,
			TasksTotal:   12,
		},
	}

	if err := store.Save(instances); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("loaded %d, want 1", len(loaded))
	}
	if loaded[0].Title != "auth" {
		t.Errorf("Title = %q", loaded[0].Title)
	}
	if loaded[0].Status != StatusRunning {
		t.Errorf("Status = %v", loaded[0].Status)
	}
	if loaded[0].TasksDone != 3 {
		t.Errorf("TasksDone = %d", loaded[0].TasksDone)
	}
}

func TestStore_LoadNonExistent(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "missing.json"))
	instances, err := store.Load()
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}
	if instances != nil {
		t.Error("should return nil for missing file")
	}
}

func TestStore_Path(t *testing.T) {
	store := NewStore("/custom/path.json")
	if store.Path() != "/custom/path.json" {
		t.Errorf("Path = %q", store.Path())
	}
}
