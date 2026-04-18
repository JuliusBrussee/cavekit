package team

import (
	"context"
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

// Manager is the high-level facade over team coordination. It composes a
// RefClient (for all git-ref CAS I/O), the identity/lease/config state on
// disk, and the claim/release/heartbeat protocol. Commands should talk to
// Manager — not to Ref or Ledger directly — so invariants stay in one place.
type Manager struct {
	Root     string
	Exec     execpkg.Executor
	Stderr   io.Writer
	Now      func() time.Time
	Hostname string
	PID      int
	Ref      *RefClient
}

type InitResult struct {
	Schema        string   `json:"schema"`
	Identity      Identity `json:"identity"`
	RosterCreated bool     `json:"roster_created"`
	Warnings      []string `json:"warnings,omitempty"`
	RefReady      bool     `json:"ref_ready"`
}

type JoinResult struct {
	Schema   string   `json:"schema"`
	Identity Identity `json:"identity"`
	Already  bool     `json:"already"`
}

type ClaimResult struct {
	Schema           string   `json:"schema"`
	Task             string   `json:"task"`
	Already          bool     `json:"already"`
	CommitSHA        string   `json:"commit_sha,omitempty"`
	ConflictingOwner string   `json:"conflicting_owner,omitempty"`
	Paths            []string `json:"paths,omitempty"`
	Provisional      bool     `json:"provisional,omitempty"`
}

type ReleaseResult struct {
	Schema      string `json:"schema"`
	Task        string `json:"task"`
	Complete    bool   `json:"complete"`
	Noop        bool   `json:"noop"`
	CommitSHA   string `json:"commit_sha,omitempty"`
	Provisional bool   `json:"provisional,omitempty"`
}

type SyncResult struct {
	Schema         string `json:"schema"`
	Fetched        bool   `json:"fetched"`
	EventCount     int    `json:"event_count"`
	OutboxPending  int    `json:"outbox_pending"`
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
		Ref:      NewRefClient(root, executor, stderr),
	}
}

// Init bootstraps team mode: scaffolds state files, resolves identity,
// installs git hygiene, creates the ledger ref branch on origin, and wires the
// pre-commit guard hook.
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

	refReady := true
	if err := m.Ref.EnsureRemoteBranch(ctx); err != nil {
		refReady = false
		warnings = append(warnings, "ledger ref bootstrap deferred: "+err.Error())
	}

	if err := InstallCommitHook(m.Root, force); err != nil {
		warnings = append(warnings, "commit-hook install failed: "+err.Error())
	}

	return InitResult{
		Schema:        Schema,
		Identity:      identity,
		RosterCreated: rosterCreated,
		Warnings:      warnings,
		RefReady:      refReady,
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
		// Refresh the local cache against the ref so this checkout sees recent state.
		_ = m.Ref.Fetch(ctx)
		_, _ = m.Ref.Read(ctx, m.Stderr)
		return JoinResult{Schema: Schema, Identity: identity, Already: true}, nil
	}

	identity, err := ResolveIdentity(ctx, m.Exec, m.Root, email, name)
	if err != nil {
		return JoinResult{}, err
	}
	if err := WriteIdentity(m.Root, identity); err != nil {
		return JoinResult{}, err
	}
	// Pull the current ledger so the new session is immediately consistent.
	_ = m.Ref.Fetch(ctx)
	_, _ = m.Ref.Read(ctx, m.Stderr)
	_ = InstallCommitHook(m.Root, false)
	return JoinResult{Schema: Schema, Identity: identity}, nil
}

// Sync refreshes the local cache from the ledger ref. It no longer touches
// the working branch — the ledger lives on refs/heads/cavekit/team, so a
// fetch+cache refresh is sufficient.
func (m *Manager) Sync(ctx context.Context, timeoutSeconds int) (SyncResult, error) {
	if !IsInitialized(m.Root) {
		return SyncResult{}, &ExitError{Code: 1, Message: "team is not initialized; run `cavekit team init` first"}
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = 10
	}
	fetchCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	if err := m.Ref.Fetch(fetchCtx); err != nil {
		return SyncResult{}, &ExitError{Code: 7, Message: err.Error()}
	}
	events, err := m.Ref.Read(ctx, m.Stderr)
	if err != nil {
		return SyncResult{}, err
	}
	// Opportunistically drain any offline-queued events now that we appear
	// online. This is what keeps manual `team sync` a useful catch-up hook.
	cfg, _ := LoadConfig(m.Root)
	_, _ = m.Ref.Publish(ctx, nil, "sync: drain outbox", cfg.AllowOffline)

	return SyncResult{
		Schema:        Schema,
		Fetched:       true,
		EventCount:    len(events),
		OutboxPending: OutboxPendingCount(m.Root),
	}, nil
}

// Claim reserves a task for the local identity using a CAS publish to the
// ledger ref. Supports optional --paths scoping. If CAS loses, we re-fetch
// and retry once; a persistent loss returns exit 5.
func (m *Manager) Claim(ctx context.Context, taskID string, paths []string) (ClaimResult, error) {
	if err := validateTaskID(taskID); err != nil {
		return ClaimResult{}, err
	}
	identity, err := m.requireIdentity()
	if err != nil {
		return ClaimResult{}, err
	}
	cfg, err := LoadConfig(m.Root)
	if err != nil {
		return ClaimResult{}, err
	}

	// Always refetch before a claim — we need the newest state to check
	// active-claim conflicts before CAS.
	_ = m.Ref.Fetch(ctx)
	if _, err := m.Ref.Read(ctx, m.Stderr); err != nil {
		return ClaimResult{}, err
	}

	return m.claimWithRetry(ctx, taskID, paths, identity, cfg, 2)
}

func (m *Manager) claimWithRetry(ctx context.Context, taskID string, paths []string, identity Identity, cfg Config, attempts int) (ClaimResult, error) {
	if attempts <= 0 {
		return ClaimResult{}, &ExitError{Code: 5, Message: "lost claim race after retries"}
	}

	selected, err := selectSite(m.Root, taskID)
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
	rawReady := site.ReadyTasks(selected, statuses)
	if !containsTask(rawReady, taskID) {
		return ClaimResult{}, &ExitError{Code: 6, Message: fmt.Sprintf("task not in frontier: %s", taskID)}
	}

	ttl := time.Duration(cfg.LeaseTTLSeconds) * time.Second
	allActive := AllActiveClaims(events, ttl, m.Now())
	for _, claim := range allActive {
		if claim.Task == taskID {
			if claim.Session == identity.Session {
				return ClaimResult{Schema: Schema, Task: taskID, Already: true, Paths: claim.Paths}, nil
			}
			return ClaimResult{}, &ExitError{
				Code:    3,
				Message: fmt.Sprintf("task claimed by another user: %s", claim.Owner),
			}
		}
		// Path-level exclusion: if the requested paths overlap another session's
		// scoped claim, refuse. Unscoped claims (len(claim.Paths)==0) don't count;
		// if *we* asked for unscoped, the task-ID check above already covers it.
		if len(paths) > 0 && len(claim.Paths) > 0 && claim.Session != identity.Session && PathsOverlap(paths, claim.Paths) {
			return ClaimResult{}, &ExitError{
				Code:    3,
				Message: fmt.Sprintf("path conflict with %s (%s on %s): %v", claim.Owner, claim.Task, claim.Paths, paths),
			}
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
		Paths:       paths,
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
		Paths:      paths,
	}
	// Append to local cache first so in-process reads see our intent, then
	// CAS-publish to the ref. If CAS loses we roll back and retry.
	if err := AppendLedgerEvent(m.Root, claimEvent); err != nil {
		_ = DeleteLease(m.Root, taskID)
		return ClaimResult{}, err
	}

	res, err := m.Ref.Publish(ctx, []LedgerEvent{claimEvent}, "claim: "+taskID, cfg.AllowOffline)
	if err != nil {
		if exitErr, ok := err.(*ExitError); ok && exitErr.Code == 5 {
			// CAS lost — roll back and retry with fresh state.
			_ = DeleteLease(m.Root, taskID)
			_ = AppendLedgerEvent(m.Root, LedgerEvent{
				TS:      m.Now().Format(time.RFC3339),
				Type:    EventRelease,
				Task:    taskID,
				Owner:   identity.Email,
				Host:    m.Hostname,
				Session: identity.Session,
				Note:    "cas-lost, retrying",
			})
			_ = m.Ref.Fetch(ctx)
			_, _ = m.Ref.Read(ctx, m.Stderr)
			return m.claimWithRetry(ctx, taskID, paths, identity, cfg, attempts-1)
		}
		_ = DeleteLease(m.Root, taskID)
		return ClaimResult{}, err
	}

	return ClaimResult{
		Schema:      Schema,
		Task:        taskID,
		CommitSHA:   res.CommitSHA,
		Paths:       paths,
		Provisional: res.Provisional,
	}, nil
}

// Release ends ownership of a task. On --complete we additionally mark it
// done in the ledger so downstream frontiers are unblocked everywhere.
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
	_ = m.Ref.Fetch(ctx)
	events, err := m.Ref.Read(ctx, m.Stderr)
	if err != nil {
		return ReleaseResult{}, err
	}
	active := ActiveClaims(events, time.Duration(cfg.LeaseTTLSeconds)*time.Second, m.Now())
	claim, ok := active[taskID]
	if !ok || claim.Session != identity.Session {
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
	res, err := m.Ref.Publish(ctx, []LedgerEvent{event}, commitMsg, cfg.AllowOffline)
	if err != nil {
		return ReleaseResult{}, err
	}
	return ReleaseResult{
		Schema:      Schema,
		Task:        taskID,
		Complete:    complete,
		CommitSHA:   res.CommitSHA,
		Provisional: res.Provisional,
	}, nil
}

// Heartbeat refreshes the lease + appends a heartbeat event locally every
// interval, and batches publishes to the ledger ref every
// HeartbeatPublishEvery ticks (so remote teammates see liveness).
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

	var pending []LedgerEvent
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
		ev := LedgerEvent{
			TS:         now.Format(time.RFC3339),
			Type:       EventHeartbeat,
			Task:       taskID,
			Owner:      identity.Email,
			Host:       m.Hostname,
			Session:    identity.Session,
			LeaseUntil: lease.ExpiresAt,
			Paths:      lease.Paths,
		}
		pending = append(pending, ev)
		return AppendLedgerEvent(m.Root, ev)
	}

	flush := func() {
		if len(pending) == 0 {
			return
		}
		commitCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		batch := pending
		pending = nil
		if _, err := m.Ref.Publish(commitCtx, batch, "heartbeat: "+taskID, true); err != nil {
			if m.Stderr != nil {
				fmt.Fprintf(m.Stderr, "heartbeat publish failed for %s: %v\n", taskID, err)
			}
			// Put events back on the pending list — we'll retry next flush. The
			// outbox also captured them if AllowOffline was allowed.
			pending = append(batch, pending...)
		}
	}

	if once {
		if err := tick(); err != nil {
			return err
		}
		flush()
		return nil
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	failures := 0
	ticksSincePublish := 0
	publishEvery := cfg.HeartbeatPublishEvery
	if publishEvery <= 0 {
		publishEvery = defaultHeartbeatPublishEvery
	}
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	for {
		select {
		case <-ctx.Done():
			flush()
			return ctx.Err()
		case <-signals:
			flush()
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
			ticksSincePublish++
			if ticksSincePublish >= publishEvery {
				flush()
				ticksSincePublish = 0
			}
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
		"repository not found",
		"operation not permitted",
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
