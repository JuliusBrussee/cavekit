package team

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"time"

	"github.com/JuliusBrussee/cavekit/internal/site"
)

// NextSuggestion is what `cavekit team next` hands back: the best unclaimed
// frontier task for the local identity given current claim topology.
type NextSuggestion struct {
	Schema      string         `json:"schema"`
	Task        *site.Task     `json:"task,omitempty"`
	Paths       []string       `json:"paths,omitempty"`
	SkippedBy   map[string]string `json:"skipped_by,omitempty"` // task_id → reason
	Alternatives []site.Task   `json:"alternatives,omitempty"`
}

// NextTask walks the ready frontier and returns the highest-priority task that
// (a) is not actively claimed, (b) has no path overlap with any active claim,
// and (c) prefers tasks matching the identity's roster focus hints.
func NextTask(root string, identity Identity, stderr io.Writer) (NextSuggestion, error) {
	cfg, err := LoadConfig(root)
	if err != nil {
		return NextSuggestion{}, err
	}
	events, err := ReadLedger(root, stderr)
	if err != nil {
		return NextSuggestion{}, err
	}
	selected, err := selectSite(root, "")
	if err != nil {
		return NextSuggestion{}, err
	}

	statuses, err := site.TrackStatus(filepath.Join(root, "context", "impl"))
	if err != nil {
		return NextSuggestion{}, err
	}
	for taskID := range CompletedTasks(events) {
		statuses[taskID] = site.TaskDone
	}

	claims := AllActiveClaims(events, time.Duration(cfg.LeaseTTLSeconds)*time.Second, time.Now().UTC())
	ready := site.ReadyTasks(selected, statuses)

	skipped := map[string]string{}
	var candidates []site.Task
	for _, task := range ready {
		blocked := false
		for _, claim := range claims {
			if claim.Task == task.ID {
				if claim.Session == identity.Session {
					// Already claimed by us — treat as the immediate suggestion.
					return NextSuggestion{Schema: Schema, Task: &task, Paths: claim.Paths}, nil
				}
				skipped[task.ID] = fmt.Sprintf("claimed by %s", claim.Owner)
				blocked = true
				break
			}
		}
		if blocked {
			continue
		}
		// Path overlap: if this task's probable file footprint conflicts with any
		// active claim, skip. We can only use hints (no task-to-paths map yet),
		// so fall back to task spec as a coarse signal — tasks in the same spec
		// often touch the same subsystem.
		conflict := false
		for _, claim := range claims {
			if claim.Session == identity.Session {
				continue
			}
			if len(claim.Paths) == 0 {
				// Unscoped claims don't block path-disjoint work.
				continue
			}
			// Heuristic: if the task's spec prefix is literally a substring of any
			// claim path, assume overlap. Callers can override by passing explicit
			// --paths to `team claim`.
			for _, p := range claim.Paths {
				if task.Spec != "" && substringMatch(p, task.Spec) {
					skipped[task.ID] = fmt.Sprintf("path overlap with %s (%s)", claim.Owner, p)
					conflict = true
					break
				}
			}
			if conflict {
				break
			}
		}
		if conflict {
			continue
		}
		candidates = append(candidates, task)
	}

	if len(candidates) == 0 {
		return NextSuggestion{Schema: Schema, SkippedBy: skipped}, nil
	}

	// Prefer tasks whose spec matches the roster focus for this identity.
	focus := rosterFocus(root, identity.Email)
	sort.SliceStable(candidates, func(i, j int) bool {
		return taskRank(candidates[i], focus) < taskRank(candidates[j], focus)
	})

	primary := candidates[0]
	alts := append([]site.Task{}, candidates[1:]...)
	return NextSuggestion{
		Schema:       Schema,
		Task:         &primary,
		SkippedBy:    skipped,
		Alternatives: alts,
	}, nil
}

// taskRank is lower-is-better: prefer lower tier, then roster-focus match,
// then task ID order for determinism.
func taskRank(task site.Task, focus []string) int {
	rank := task.Tier * 100
	if !matchesFocus(task, focus) {
		rank += 10
	}
	return rank
}

func matchesFocus(task site.Task, focus []string) bool {
	if len(focus) == 0 {
		return false
	}
	for _, hint := range focus {
		if hint == "" {
			continue
		}
		if substringMatch(task.Spec, hint) || substringMatch(task.Title, hint) {
			return true
		}
	}
	return false
}

func substringMatch(haystack, needle string) bool {
	if haystack == "" || needle == "" {
		return false
	}
	return containsFold(haystack, needle)
}
