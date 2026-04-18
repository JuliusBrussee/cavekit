package site

import (
	"fmt"
	"strings"
)

// ClaimInfo is the minimal claim state needed to filter a ready frontier.
type ClaimInfo struct {
	Owner   string
	Session string
}

// ReadyTasks returns all tasks that are ready to execute:
// not yet done, and all blockedBy dependencies are complete.
func ReadyTasks(s *Site, completed TaskStatusMap) []Task {
	doneSet := make(map[string]bool)
	for id, status := range completed {
		if status == TaskDone {
			doneSet[id] = true
		}
	}

	var ready []Task
	for _, t := range s.Tasks {
		if doneSet[t.ID] {
			continue
		}
		allDepsMet := true
		for _, dep := range t.BlockedBy {
			if dep == "" {
				continue
			}
			if !doneSet[dep] {
				allDepsMet = false
				break
			}
		}
		if allDepsMet {
			ready = append(ready, t)
		}
	}
	return ready
}

// FilterReadyTasks removes tasks held by other claims from a ready frontier.
// The current owner's currently active task remains visible.
func FilterReadyTasks(ready []Task, active map[string]ClaimInfo, currentOwner, currentSession, currentTask string) ([]Task, []string) {
	if len(active) == 0 {
		return ready, nil
	}

	filtered := make([]Task, 0, len(ready))
	excluded := make([]string, 0)
	for _, task := range ready {
		claim, ok := active[task.ID]
		if !ok {
			filtered = append(filtered, task)
			continue
		}
		if claim.Owner != "" && claim.Owner != currentOwner {
			excluded = append(excluded, task.ID)
			continue
		}
		if claim.Owner == currentOwner && claim.Session == currentSession && task.ID == currentTask {
			filtered = append(filtered, task)
			continue
		}
		excluded = append(excluded, task.ID)
	}
	return filtered, excluded
}

// FrontierSummary returns a human-readable summary of ready tasks.
func FrontierSummary(ready []Task) string {
	if len(ready) == 0 {
		return "No tasks ready (all done or blocked)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d task(s) ready for parallel execution:\n", len(ready))
	for _, t := range ready {
		deps := "none"
		if len(t.BlockedBy) > 0 {
			filtered := filterEmpty(t.BlockedBy)
			if len(filtered) > 0 {
				deps = strings.Join(filtered, ", ")
			}
		}
		fmt.Fprintf(&b, "  %s: %s (tier %d, deps: %s, effort: %s)\n",
			t.ID, t.Title, t.Tier, deps, t.Effort)
	}
	return b.String()
}

func filterEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
