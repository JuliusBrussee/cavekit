package site

import (
	"strings"
	"testing"
)

func TestReadyTasks_AllTier0Ready(t *testing.T) {
	s := &Site{
		Tasks: []Task{
			{ID: "T-001", Title: "Setup", Tier: 0},
			{ID: "T-002", Title: "Config", Tier: 0},
			{ID: "T-003", Title: "Auth", Tier: 1, BlockedBy: []string{"T-001"}},
		},
	}
	ready := ReadyTasks(s, TaskStatusMap{})
	if len(ready) != 2 {
		t.Errorf("expected 2 ready tasks, got %d", len(ready))
	}
}

func TestReadyTasks_Tier1UnlockedAfterDeps(t *testing.T) {
	s := &Site{
		Tasks: []Task{
			{ID: "T-001", Title: "Setup", Tier: 0},
			{ID: "T-002", Title: "Config", Tier: 0},
			{ID: "T-003", Title: "Auth", Tier: 1, BlockedBy: []string{"T-001"}},
		},
	}
	completed := TaskStatusMap{"T-001": TaskDone, "T-002": TaskDone}
	ready := ReadyTasks(s, completed)
	if len(ready) != 1 {
		t.Errorf("expected 1 ready task, got %d", len(ready))
	}
	if ready[0].ID != "T-003" {
		t.Errorf("expected T-003, got %s", ready[0].ID)
	}
}

func TestReadyTasks_BlockedStaysBlocked(t *testing.T) {
	s := &Site{
		Tasks: []Task{
			{ID: "T-001", Title: "Setup", Tier: 0},
			{ID: "T-002", Title: "Auth", Tier: 1, BlockedBy: []string{"T-001"}},
			{ID: "T-003", Title: "UI", Tier: 2, BlockedBy: []string{"T-001", "T-002"}},
		},
	}
	completed := TaskStatusMap{"T-001": TaskDone}
	ready := ReadyTasks(s, completed)
	if len(ready) != 1 {
		t.Errorf("expected 1 ready task (T-002), got %d", len(ready))
	}
	if len(ready) > 0 && ready[0].ID != "T-002" {
		t.Errorf("expected T-002, got %s", ready[0].ID)
	}
}

func TestReadyTasks_AllDone(t *testing.T) {
	s := &Site{
		Tasks: []Task{
			{ID: "T-001", Title: "Setup", Tier: 0},
			{ID: "T-002", Title: "Config", Tier: 0},
		},
	}
	completed := TaskStatusMap{"T-001": TaskDone, "T-002": TaskDone}
	ready := ReadyTasks(s, completed)
	if len(ready) != 0 {
		t.Errorf("expected 0 ready tasks, got %d", len(ready))
	}
}

func TestReadyTasks_EmptyBlockedByIgnored(t *testing.T) {
	s := &Site{
		Tasks: []Task{
			{ID: "T-001", Title: "Setup", Tier: 0, BlockedBy: []string{""}},
		},
	}
	ready := ReadyTasks(s, TaskStatusMap{})
	if len(ready) != 1 {
		t.Errorf("expected 1 ready task (empty dep ignored), got %d", len(ready))
	}
}

func TestReadyTasks_MultipleDepsPartiallyMet(t *testing.T) {
	s := &Site{
		Tasks: []Task{
			{ID: "T-001", Title: "A", Tier: 0},
			{ID: "T-002", Title: "B", Tier: 0},
			{ID: "T-003", Title: "C", Tier: 1, BlockedBy: []string{"T-001", "T-002"}},
		},
	}
	// Only T-001 done — T-003 should still be blocked
	completed := TaskStatusMap{"T-001": TaskDone}
	ready := ReadyTasks(s, completed)
	// T-002 is ready (no deps), T-003 is blocked (T-002 not done)
	if len(ready) != 1 {
		t.Errorf("expected 1 ready task (T-002), got %d", len(ready))
	}
	if len(ready) > 0 && ready[0].ID != "T-002" {
		t.Errorf("expected T-002, got %s", ready[0].ID)
	}
}

func TestFrontierSummary_Empty(t *testing.T) {
	result := FrontierSummary(nil)
	if !strings.Contains(result, "No tasks ready") {
		t.Errorf("expected 'No tasks ready', got: %s", result)
	}
}

func TestFrontierSummary_WithTasks(t *testing.T) {
	tasks := []Task{
		{ID: "T-001", Title: "Setup", Tier: 0, Effort: "S"},
		{ID: "T-002", Title: "Config", Tier: 0, Effort: "M"},
	}
	result := FrontierSummary(tasks)
	if !strings.Contains(result, "2 task(s)") {
		t.Errorf("expected '2 task(s)', got: %s", result)
	}
	if !strings.Contains(result, "T-001") || !strings.Contains(result, "T-002") {
		t.Errorf("expected both task IDs in summary, got: %s", result)
	}
}

func TestFilterReadyTasks_ExcludesOtherOwners(t *testing.T) {
	ready := []Task{
		{ID: "T-001", Title: "One"},
		{ID: "T-002", Title: "Two"},
	}
	active := map[string]ClaimInfo{
		"T-001": {Owner: "other@example.com", Session: "other"},
		"T-002": {Owner: "me@example.com", Session: "mine"},
	}

	filtered, excluded := FilterReadyTasks(ready, active, "me@example.com", "mine", "T-002")
	if len(filtered) != 1 || filtered[0].ID != "T-002" {
		t.Fatalf("expected only current owner's active task to remain, got %+v", filtered)
	}
	if len(excluded) != 1 || excluded[0] != "T-001" {
		t.Fatalf("expected T-001 to be excluded, got %+v", excluded)
	}
}
