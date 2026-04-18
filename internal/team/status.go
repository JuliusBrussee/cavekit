package team

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JuliusBrussee/cavekit/internal/site"
	"github.com/JuliusBrussee/cavekit/internal/worktree"
)

type ActivityRow struct {
	RelativeTime string `json:"relative_time"`
	Owner        string `json:"owner"`
	Type         string `json:"type"`
	Task         string `json:"task"`
}

type ActiveClaimRow struct {
	Owner      string `json:"owner"`
	Task       string `json:"task"`
	AcquiredAt string `json:"acquired_at"`
	Stale      bool   `json:"stale"`
}

type StatusReport struct {
	Schema           string           `json:"schema"`
	Site             string           `json:"site,omitempty"`
	FrontierRaw      []string         `json:"frontier_raw"`
	FrontierFiltered []string         `json:"frontier_filtered"`
	ExcludedByTeam   []string         `json:"excluded_by_team"`
	ActiveClaims     []ActiveClaimRow `json:"active_claims"`
	RecentActivity   []ActivityRow    `json:"recent_activity"`
	IdleMembers      []string         `json:"idle_members"`
	Conflicts        []ConflictRow    `json:"conflicts,omitempty"`
	OutboxPending    int              `json:"outbox_pending,omitempty"`
}

// ConflictRow summarizes a recent race or override for operator awareness.
// Surfaced by `team status --conflicts`.
type ConflictRow struct {
	RelativeTime string `json:"relative_time"`
	Type         string `json:"type"` // cas-lost, path-override, stolen-stale, rollback
	Owner        string `json:"owner"`
	Task         string `json:"task"`
	Note         string `json:"note"`
}

// CollectConflicts scans the ledger for recent non-standard events: rollbacks,
// stolen stale leases, commit-guard overrides, and notes containing "race".
// The caller decides whether to include these in status output.
func CollectConflicts(root string, stderr io.Writer) []ConflictRow {
	events, err := ReadLedger(root, stderr)
	if err != nil {
		return nil
	}
	rows := make([]ConflictRow, 0, 8)
	now := time.Now().UTC()
	for i := len(events) - 1; i >= 0 && len(rows) < 15; i-- {
		e := events[i]
		note := strings.ToLower(e.Note)
		typ := ""
		switch {
		case e.Type == EventNote && strings.Contains(note, "commit-guard override"):
			typ = "path-override"
		case strings.Contains(note, "rolled back"):
			typ = "rollback"
		case strings.Contains(note, "cas-lost") || strings.Contains(note, "lost claim race"):
			typ = "cas-lost"
		case strings.Contains(note, "stolen stale"):
			typ = "stolen-stale"
		}
		if typ == "" {
			continue
		}
		rows = append(rows, ConflictRow{
			RelativeTime: relativeTime(now, parseEventTime(e.TS)),
			Type:         typ,
			Owner:        e.Owner,
			Task:         e.Task,
			Note:         e.Note,
		})
	}
	return rows
}

// FormatConflicts renders the conflict tail + outbox summary as plain text.
func FormatConflicts(report StatusReport) string {
	var b strings.Builder
	b.WriteString("\nConflicts & Races\n")
	if len(report.Conflicts) == 0 {
		b.WriteString("—\n")
	} else {
		b.WriteString("time | type | owner | task | note\n")
		for _, row := range report.Conflicts {
			fmt.Fprintf(&b, "%s | %s | %s | %s | %s\n",
				row.RelativeTime, row.Type, row.Owner, row.Task, row.Note)
		}
	}
	if report.OutboxPending > 0 {
		fmt.Fprintf(&b, "\nOutbox: %d provisional event(s) queued offline\n", report.OutboxPending)
	}
	return b.String()
}

func BuildStatusReport(root string, identity *Identity, taskFilter, userFilter string, stderr io.Writer) (StatusReport, error) {
	cfg, err := LoadConfig(root)
	if err != nil {
		return StatusReport{}, err
	}
	events, err := ReadLedger(root, stderr)
	if err != nil {
		return StatusReport{}, err
	}
	selectedSite, err := selectSite(root, taskFilter)
	if err != nil {
		return StatusReport{}, err
	}

	statuses, err := site.TrackStatus(filepath.Join(root, "context", "impl"))
	if err != nil {
		return StatusReport{}, err
	}
	for taskID := range CompletedTasks(events) {
		statuses[taskID] = site.TaskDone
	}

	ready := site.ReadyTasks(selectedSite, statuses)
	active := ActiveClaims(events, time.Duration(cfg.LeaseTTLSeconds)*time.Second, time.Now().UTC())
	claimInfo := make(map[string]site.ClaimInfo, len(active))
	for taskID, claim := range active {
		claimInfo[taskID] = site.ClaimInfo{Owner: claim.Owner, Session: claim.Session}
	}

	currentOwner, currentSession, currentTask := "", "", ""
	if identity != nil {
		currentOwner = identity.Email
		currentSession = identity.Session
	}
	filteredReady, excluded := site.FilterReadyTasks(ready, claimInfo, currentOwner, currentSession, currentTask)

	report := StatusReport{
		Schema:           Schema,
		Site:             selectedSite.Path,
		FrontierRaw:      taskIDs(ready),
		FrontierFiltered: taskIDs(filteredReady),
		ExcludedByTeam:   excluded,
		ActiveClaims:     buildActiveClaimRows(active, userFilter),
		RecentActivity:   buildRecentActivity(events, userFilter, taskFilter),
		IdleMembers:      buildIdleMembers(root, active, userFilter),
	}

	if taskFilter != "" {
		report.FrontierRaw = filterTaskIDs(report.FrontierRaw, taskFilter)
		report.FrontierFiltered = filterTaskIDs(report.FrontierFiltered, taskFilter)
		report.ExcludedByTeam = filterTaskIDs(report.ExcludedByTeam, taskFilter)
		report.ActiveClaims = filterActiveClaims(report.ActiveClaims, taskFilter)
	}
	if userFilter != "" {
		report.ActiveClaims = filterActiveClaimsByOwner(report.ActiveClaims, userFilter)
		report.RecentActivity = filterActivityByOwner(report.RecentActivity, userFilter)
		report.IdleMembers = filterOwners(report.IdleMembers, userFilter)
	}
	return report, nil
}

func FormatStatusReport(report StatusReport) string {
	var b strings.Builder

	b.WriteString("Active Claims\n")
	if len(report.ActiveClaims) == 0 {
		b.WriteString("owner | task | acquired-at | stale?\n")
		b.WriteString("— | — | — | —\n")
	} else {
		b.WriteString("owner | task | acquired-at | stale?\n")
		for _, row := range report.ActiveClaims {
			stale := "no"
			if row.Stale {
				stale = "yes"
			}
			fmt.Fprintf(&b, "%s | %s | %s | %s\n", row.Owner, row.Task, row.AcquiredAt, stale)
		}
	}

	b.WriteString("\nRecent Activity\n")
	b.WriteString("time | owner | type | task\n")
	for _, row := range report.RecentActivity {
		fmt.Fprintf(&b, "%s | %s | %s | %s\n", row.RelativeTime, row.Owner, row.Type, row.Task)
	}
	if len(report.RecentActivity) == 0 {
		b.WriteString("— | — | — | —\n")
	}

	b.WriteString("\nIdle Members\n")
	if len(report.IdleMembers) == 0 {
		b.WriteString("—\n")
	} else {
		for _, owner := range report.IdleMembers {
			b.WriteString(owner + "\n")
		}
	}
	return b.String()
}

func selectSite(root, taskFilter string) (*site.Site, error) {
	files, err := site.Discover(root)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no build site found under context/plans, context/sites, or context/frontiers")
	}

	if taskFilter != "" {
		var matches []*site.Site
		for _, file := range files {
			parsed, err := site.Parse(file.Path)
			if err != nil {
				continue
			}
			if parsed.TaskByID(taskFilter) != nil {
				matches = append(matches, parsed)
			}
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("task %s does not exist in any discovered build site", taskFilter)
		}
		if len(matches) > 1 {
			return nil, fmt.Errorf("task %s appears in multiple build sites", taskFilter)
		}
		return matches[0], nil
	}

	worktrees, _ := worktree.DiscoverAll(root)
	worktreeMap := map[string]worktree.DiscoveredWorktree{}
	for _, wt := range worktrees {
		worktreeMap[wt.SiteName] = wt
	}
	ranked, err := site.RankAndSelect(files, nil, "", func(name string) (bool, bool) {
		wt, ok := worktreeMap[name]
		if !ok {
			return false, false
		}
		return true, wt.HasRalphLoop
	})
	if err != nil {
		return nil, err
	}
	for _, candidate := range ranked {
		if candidate.Selected {
			return site.Parse(candidate.File.Path)
		}
	}
	return site.Parse(files[0].Path)
}

func taskIDs(tasks []site.Task) []string {
	out := make([]string, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, task.ID)
	}
	return out
}

func buildActiveClaimRows(active map[string]ActiveClaim, userFilter string) []ActiveClaimRow {
	rows := make([]ActiveClaimRow, 0, len(active))
	for _, claim := range active {
		if userFilter != "" && claim.Owner != normalizeEmail(userFilter) {
			continue
		}
		rows = append(rows, ActiveClaimRow{
			Owner:      claim.Owner,
			Task:       claim.Task,
			AcquiredAt: claim.AcquiredAt,
			Stale:      claim.Stale,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].AcquiredAt != rows[j].AcquiredAt {
			return rows[i].AcquiredAt < rows[j].AcquiredAt
		}
		if rows[i].Owner != rows[j].Owner {
			return rows[i].Owner < rows[j].Owner
		}
		return rows[i].Task < rows[j].Task
	})
	return rows
}

func buildRecentActivity(events []LedgerEvent, userFilter, taskFilter string) []ActivityRow {
	rows := make([]ActivityRow, 0, 10)
	now := time.Now().UTC()
	for i := len(events) - 1; i >= 0 && len(rows) < 10; i-- {
		event := events[i]
		if userFilter != "" && event.Owner != normalizeEmail(userFilter) {
			continue
		}
		if taskFilter != "" && event.Task != "" && event.Task != taskFilter {
			continue
		}
		task := event.Task
		if task == "" {
			task = "—"
		}
		rows = append(rows, ActivityRow{
			RelativeTime: relativeTime(now, parseEventTime(event.TS)),
			Owner:        event.Owner,
			Type:         string(event.Type),
			Task:         task,
		})
	}
	return rows
}

func buildIdleMembers(root string, active map[string]ActiveClaim, userFilter string) []string {
	members, err := ReadRosterMembers(root)
	if err != nil {
		return nil
	}
	activeOwners := map[string]bool{}
	for _, claim := range active {
		activeOwners[claim.Owner] = true
	}

	idle := make([]string, 0, len(members))
	for _, member := range members {
		if userFilter != "" && member.Identity != normalizeEmail(userFilter) {
			continue
		}
		if activeOwners[member.Identity] {
			continue
		}
		idle = append(idle, member.Identity)
	}
	sort.Strings(idle)
	return idle
}

func relativeTime(now, then time.Time) string {
	if then.IsZero() {
		return "—"
	}
	delta := now.Sub(then)
	switch {
	case delta < time.Minute:
		return fmt.Sprintf("%ds ago", int(delta.Seconds()))
	case delta < time.Hour:
		return fmt.Sprintf("%dm ago", int(delta.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(delta.Hours()))
	}
}

func filterTaskIDs(ids []string, taskID string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == taskID {
			out = append(out, id)
		}
	}
	return out
}

func filterActiveClaims(rows []ActiveClaimRow, taskID string) []ActiveClaimRow {
	out := make([]ActiveClaimRow, 0, len(rows))
	for _, row := range rows {
		if row.Task == taskID {
			out = append(out, row)
		}
	}
	return out
}

func filterActiveClaimsByOwner(rows []ActiveClaimRow, owner string) []ActiveClaimRow {
	owner = normalizeEmail(owner)
	out := make([]ActiveClaimRow, 0, len(rows))
	for _, row := range rows {
		if row.Owner == owner {
			out = append(out, row)
		}
	}
	return out
}

func filterActivityByOwner(rows []ActivityRow, owner string) []ActivityRow {
	owner = normalizeEmail(owner)
	out := make([]ActivityRow, 0, len(rows))
	for _, row := range rows {
		if row.Owner == owner {
			out = append(out, row)
		}
	}
	return out
}

func filterOwners(owners []string, owner string) []string {
	owner = normalizeEmail(owner)
	out := make([]string, 0, len(owners))
	for _, candidate := range owners {
		if candidate == owner {
			out = append(out, candidate)
		}
	}
	return out
}
