package team

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	execpkg "github.com/JuliusBrussee/cavekit/internal/exec"
	"github.com/JuliusBrussee/cavekit/internal/site"
)

type Manager struct {
	Root     string
	Exec     execpkg.Executor
	Stderr   io.Writer
	Now      func() time.Time
	Hostname string
	PID      int
}

type InitResult struct {
	Schema        string   `json:"schema"`
	Identity      Identity `json:"identity"`
	RosterCreated bool     `json:"roster_created"`
	Warnings      []string `json:"warnings,omitempty"`
}

type JoinResult struct {
	Schema  string   `json:"schema"`
	Identity Identity `json:"identity"`
	Already bool     `json:"already"`
}

type ClaimResult struct {
	Schema       string `json:"schema"`
	Task         string `json:"task"`
	Already      bool   `json:"already"`
	CommitSHA    string `json:"commit_sha,omitempty"`
	ConflictingOwner string `json:"conflicting_owner,omitempty"`
}

type ReleaseResult struct {
	Schema   string `json:"schema"`
	Task     string `json:"task"`
	Complete bool   `json:"complete"`
	Noop     bool   `json:"noop"`
	CommitSHA string `json:"commit_sha,omitempty"`
}

type SyncResult struct {
	Schema     string `json:"schema"`
	Fetched    bool   `json:"fetched"`
	EventCount int    `json:"event_count"`
}

func NewManager(root string, executor execpkg.Executor, stderr io.Writer) *Manager {
	host, _ := os.Hostname()
	return &Manager{
		Root:     root,
		Exec:     executor,
		Stderr:   stderr,
		Now:      func() time.Time { return time.Now().UTC() },
		Hostname: host,
		PID:      os.Getpid(),
	}
}

func (m *Manager) Init(ctx context.Context, force bool, email, name string) (InitResult, error) {
	if IsInitialized(m.Root) && !force {
		return InitResult{}, &ExitError{Code: 1, Message: "team already initialized; re-run with --force to rewrite scaffolding"}
	}

	if err := EnsureLedger(m.Root); err != nil {
		return InitResult{}, err
	}
	if err := WriteDefaultConfig(m.Root); err != nil {
		return InitResult{}, err
	}

	identity, err := ResolveIdentity(ctx, m.Exec, m.Root, email, name)
	if err != nil {
		return InitResult{}, err
	}
	if err := WriteIdentity(m.Root, identity); err != nil {
		return InitResult{}, err
	}

	rosterCreated, err := EnsureRoster(m.Root)
	if err != nil {
		return InitResult{}, err
	}
	if err := EnsureGitignoreBlock(m.Root, force); err != nil {
		return InitResult{}, err
	}
	warnings, err := EnsureGitattributesBlock(m.Root, force)
	if err != nil {
		return InitResult{}, err
	}

	return InitResult{
		Schema:        Schema,
		Identity:      identity,
		RosterCreated: rosterCreated,
		Warnings:      warnings,
	}, nil
}

func (m *Manager) Join(ctx context.Context, email, name string, strict bool) (JoinResult, error) {
	if !IsInitialized(m.Root) {
		return JoinResult{}, &ExitError{Code: 1, Message: "team is not initialized; run `cavekit team init` first"}
	}
	if fileExists(IdentityPath(m.Root)) {
		identity, err := ReadIdentity(m.Root)
		if err != nil {
			return JoinResult{}, err
		}
		if strict {
			return JoinResult{}, &ExitError{Code: 3, Message: "already joined"}
		}
		return JoinResult{Schema: Schema, Identity: identity, Already: true}, nil
	}

	identity, err := ResolveIdentity(ctx, m.Exec, m.Root, email, name)
	if err != nil {
		return JoinResult{}, err
	}
	if err := WriteIdentity(m.Root, identity); err != nil {
		return JoinResult{}, err
	}
	return JoinResult{Schema: Schema, Identity: identity}, nil
}

func (m *Manager) Sync(ctx context.Context, timeoutSeconds int) (SyncResult, error) {
	if !IsInitialized(m.Root) {
		return SyncResult{}, &ExitError{Code: 1, Message: "team is not initialized; run `cavekit team init` first"}
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = 10
	}
	fetchCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	res, err := m.Exec.RunDir(fetchCtx, m.Root, "git", "fetch")
	if err != nil {
		return SyncResult{}, err
	}
	if res.ExitCode != 0 {
		return SyncResult{}, &ExitError{Code: 7, Message: strings.TrimSpace(res.Stderr)}
	}

	events, err := ReadLedger(m.Root, m.Stderr)
	if err != nil {
		return SyncResult{}, err
	}
	return SyncResult{Schema: Schema, Fetched: true, EventCount: len(events)}, nil
}

func (m *Manager) Claim(ctx context.Context, taskID string) (ClaimResult, error) {
	if err := validateTaskID(taskID); err != nil {
		return ClaimResult{}, err
	}
	identity, err := m.requireIdentity()
	if err != nil {
		return ClaimResult{}, err
	}
	if err := m.bestEffortPullFF(ctx); err != nil && m.Stderr != nil {
		fmt.Fprintf(m.Stderr, "warning: git pull --ff-only failed: %v\n", err)
	}

	selectedSite, err := selectSite(m.Root, taskID)
	if err != nil {
		return ClaimResult{}, err
	}
	statuses, err := site.TrackStatus(filepath.Join(m.Root, "context", "impl"))
	if err != nil {
		return ClaimResult{}, err
	}
	events, err := ReadLedger(m.Root, m.Stderr)
	if err != nil {
		return ClaimResult{}, err
	}
	for doneTask := range CompletedTasks(events) {
		statuses[doneTask] = site.TaskDone
	}
	rawReady := site.ReadyTasks(selectedSite, statuses)
	if !containsTask(rawReady, taskID) {
		return ClaimResult{}, &ExitError{Code: 6, Message: fmt.Sprintf("task not in frontier: %s", taskID)}
	}

	cfg, err := LoadConfig(m.Root)
	if err != nil {
		return ClaimResult{}, err
	}
	ttl := time.Duration(cfg.LeaseTTLSeconds) * time.Second
	active := ActiveClaims(events, ttl, m.Now())
	if claim, ok := active[taskID]; ok {
		if claim.Owner == identity.Email {
			return ClaimResult{Schema: Schema, Task: taskID, Already: true}, nil
		}
		return ClaimResult{}, &ExitError{
			Code:    3,
			Message: fmt.Sprintf("task claimed by another user: %s", claim.Owner),
		}
	}

	now := m.Now()
	lease := Lease{
		Owner:       identity.Email,
		Host:        m.Hostname,
		PID:         m.PID,
		Session:     identity.Session,
		AcquiredAt:  now.Format(time.RFC3339),
		HeartbeatAt: now.Format(time.RFC3339),
		ExpiresAt:   now.Add(ttl).Format(time.RFC3339),
	}
	createRes, err := TryCreateLease(m.Root, taskID, lease, now, ttl)
	if err != nil {
		return ClaimResult{}, err
	}
	if !createRes.Created {
		if createRes.Fresh {
			return ClaimResult{}, &ExitError{Code: 4, Message: "task locally leased by another session"}
		}
		if createRes.Existing != nil {
			stolenNote := fmt.Sprintf("stolen stale %s", createRes.Existing.Owner)
			_ = AppendLedgerEvent(m.Root, LedgerEvent{
				TS:      now.Format(time.RFC3339),
				Type:    EventRelease,
				Task:    taskID,
				Owner:   normalizeEmail(createRes.Existing.Owner),
				Host:    createRes.Existing.Host,
				Session: createRes.Existing.Session,
				Note:    stolenNote,
			})
		}
		if err := DeleteLease(m.Root, taskID); err != nil {
			return ClaimResult{}, err
		}
		createRes, err = TryCreateLease(m.Root, taskID, lease, now, ttl)
		if err != nil {
			return ClaimResult{}, err
		}
		if !createRes.Created {
			return ClaimResult{}, &ExitError{Code: 4, Message: "task locally leased by another session"}
		}
	}

	claimEvent := LedgerEvent{
		TS:         now.Format(time.RFC3339),
		Type:       EventClaim,
		Task:       taskID,
		Owner:      identity.Email,
		Host:       m.Hostname,
		Session:    identity.Session,
		LeaseUntil: now.Add(ttl).Format(time.RFC3339),
	}
	if err := AppendLedgerEvent(m.Root, claimEvent); err != nil {
		_ = DeleteLease(m.Root, taskID)
		return ClaimResult{}, err
	}

	sha, err := m.commitLedger(ctx, "claim: "+taskID)
	if err != nil {
		return ClaimResult{}, err
	}
	if err := m.pushWithRaceRecovery(ctx, taskID, identity, claimEvent); err != nil {
		return ClaimResult{}, err
	}

	return ClaimResult{Schema: Schema, Task: taskID, CommitSHA: sha}, nil
}

func (m *Manager) Release(ctx context.Context, taskID, note string, complete bool) (ReleaseResult, error) {
	if err := validateTaskID(taskID); err != nil {
		return ReleaseResult{}, err
	}
	identity, err := m.requireIdentity()
	if err != nil {
		return ReleaseResult{}, err
	}
	cfg, err := LoadConfig(m.Root)
	if err != nil {
		return ReleaseResult{}, err
	}
	events, err := ReadLedger(m.Root, m.Stderr)
	if err != nil {
		return ReleaseResult{}, err
	}
	active := ActiveClaims(events, time.Duration(cfg.LeaseTTLSeconds)*time.Second, m.Now())
	claim, ok := active[taskID]
	if !ok || claim.Owner != identity.Email {
		if complete {
			return ReleaseResult{}, &ExitError{Code: 6, Message: "cannot complete unclaimed task"}
		}
		return ReleaseResult{Schema: Schema, Task: taskID, Noop: true}, nil
	}

	eventType := EventRelease
	commitMsg := "release: " + taskID
	if complete {
		eventType = EventComplete
		commitMsg = "complete: " + taskID
	}
	event := LedgerEvent{
		TS:      m.Now().Format(time.RFC3339),
		Type:    eventType,
		Task:    taskID,
		Owner:   identity.Email,
		Host:    m.Hostname,
		Session: identity.Session,
		Note:    strings.TrimSpace(note),
	}
	if err := AppendLedgerEvent(m.Root, event); err != nil {
		return ReleaseResult{}, err
	}
	if err := DeleteLease(m.Root, taskID); err != nil {
		return ReleaseResult{}, err
	}
	sha, err := m.commitLedger(ctx, commitMsg)
	if err != nil {
		return ReleaseResult{}, err
	}
	_, _ = m.Exec.RunDir(ctx, m.Root, "git", "push")
	return ReleaseResult{Schema: Schema, Task: taskID, Complete: complete, CommitSHA: sha}, nil
}

func (m *Manager) Heartbeat(ctx context.Context, taskID string, interval time.Duration, once bool) error {
	if err := validateTaskID(taskID); err != nil {
		return err
	}
	identity, err := m.requireIdentity()
	if err != nil {
		return err
	}
	cfg, err := LoadConfig(m.Root)
	if err != nil {
		return err
	}
	if interval <= 0 {
		interval = time.Duration(cfg.HeartbeatIntervalSeconds) * time.Second
	}
	tick := func() error {
		lease, err := ReadLease(m.Root, taskID)
		if err != nil {
			return err
		}
		now := m.Now()
		lease.HeartbeatAt = now.Format(time.RFC3339)
		lease.ExpiresAt = now.Add(time.Duration(cfg.LeaseTTLSeconds) * time.Second).Format(time.RFC3339)
		if err := WriteLease(m.Root, taskID, lease); err != nil {
			return err
		}
		return AppendLedgerEvent(m.Root, LedgerEvent{
			TS:         now.Format(time.RFC3339),
			Type:       EventHeartbeat,
			Task:       taskID,
			Owner:      identity.Email,
			Host:       m.Hostname,
			Session:    identity.Session,
			LeaseUntil: lease.ExpiresAt,
		})
	}

	if once {
		return tick()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	failures := 0
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-signals:
			return nil
		case <-ticker.C:
			if err := tick(); err != nil {
				failures++
				if m.Stderr != nil {
					fmt.Fprintf(m.Stderr, "heartbeat failed for %s: %v\n", taskID, err)
				}
				if failures >= 2 {
					_, _ = m.Release(ctx, taskID, "heartbeat failure", false)
					return err
				}
				continue
			}
			failures = 0
		}
	}
}

func (m *Manager) requireIdentity() (Identity, error) {
	if !fileExists(IdentityPath(m.Root)) {
		return Identity{}, &ExitError{
			Code:    1,
			Message: "identity.json missing; run `cavekit team join` first",
		}
	}
	return ReadIdentity(m.Root)
}

func (m *Manager) bestEffortPullFF(ctx context.Context) error {
	res, err := m.Exec.RunDir(ctx, m.Root, "git", "pull", "--ff-only")
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return errors.New(strings.TrimSpace(res.Stderr))
	}
	return nil
}

func (m *Manager) commitLedger(ctx context.Context, message string) (string, error) {
	res, err := m.Exec.RunDir(ctx, m.Root, "git", "add", LedgerPath(m.Root))
	if err != nil {
		return "", err
	}
	if res.ExitCode != 0 {
		return "", errors.New(strings.TrimSpace(res.Stderr))
	}

	res, err = m.Exec.RunDir(ctx, m.Root, "git", "commit", "-m", message, "--allow-empty")
	if err != nil {
		return "", err
	}
	if res.ExitCode != 0 {
		return "", errors.New(strings.TrimSpace(res.Stderr))
	}

	res, err = m.Exec.RunDir(ctx, m.Root, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	if res.ExitCode != 0 {
		return "", errors.New(strings.TrimSpace(res.Stderr))
	}
	return strings.TrimSpace(res.Stdout), nil
}

func (m *Manager) pushWithRaceRecovery(ctx context.Context, taskID string, identity Identity, claimEvent LedgerEvent) error {
	pushRes, err := m.Exec.RunDir(ctx, m.Root, "git", "push")
	if err != nil {
		return err
	}
	if pushRes.ExitCode == 0 {
		return nil
	}
	if isLocalOnlyPushFailure(pushRes.Stderr) {
		if m.Stderr != nil {
			fmt.Fprintf(m.Stderr, "warning: git push skipped for local/offline claim: %s\n", strings.TrimSpace(pushRes.Stderr))
		}
		return nil
	}

	rebaseRes, err := m.Exec.RunDir(ctx, m.Root, "git", "pull", "--rebase")
	if err != nil {
		return err
	}
	if rebaseRes.ExitCode != 0 {
		if isLocalOnlyPushFailure(rebaseRes.Stderr) {
			if m.Stderr != nil {
				fmt.Fprintf(m.Stderr, "warning: git pull --rebase skipped after push failure: %s\n", strings.TrimSpace(rebaseRes.Stderr))
			}
			return nil
		}
		return &ExitError{Code: 5, Message: "lost claim race on push"}
	}

	cfg, err := LoadConfig(m.Root)
	if err != nil {
		return err
	}
	events, err := ReadLedger(m.Root, m.Stderr)
	if err != nil {
		return err
	}
	active := ActiveClaims(events, time.Duration(cfg.LeaseTTLSeconds)*time.Second, m.Now())
	if winner, ok := active[taskID]; ok && winner.Owner != identity.Email && winner.AcquiredAt <= claimEvent.TS {
		_ = AppendLedgerEvent(m.Root, LedgerEvent{
			TS:      m.Now().Format(time.RFC3339),
			Type:    EventRelease,
			Task:    taskID,
			Owner:   identity.Email,
			Host:    m.Hostname,
			Session: identity.Session,
			Note:    "lost claim race",
		})
		_ = DeleteLease(m.Root, taskID)
		_, _ = m.commitLedger(ctx, "release: "+taskID)
		return &ExitError{Code: 5, Message: "lost claim race on push"}
	}

	pushRes, err = m.Exec.RunDir(ctx, m.Root, "git", "push")
	if err != nil {
		return err
	}
	if pushRes.ExitCode != 0 {
		return &ExitError{Code: 5, Message: "lost claim race on push"}
	}
	return nil
}

func isLocalOnlyPushFailure(stderr string) bool {
	msg := strings.ToLower(stderr)
	for _, needle := range []string{
		"has no upstream branch",
		"no configured push destination",
		"there is no tracking information",
		"could not read from remote repository",
		"couldn't find remote ref",
		"unable to access",
		"network is unreachable",
		"connection timed out",
		"does not appear to be a git repository",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func containsTask(tasks []site.Task, taskID string) bool {
	for _, task := range tasks {
		if task.ID == taskID {
			return true
		}
	}
	return false
}
