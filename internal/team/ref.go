package team

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	execpkg "github.com/JuliusBrussee/cavekit/internal/exec"
)

// The team ledger lives on its own orphan branch so it never collides with
// feature work on main. Appends go through a compare-and-swap (CAS) push that
// uses --force-with-lease; a lost race is detected in ~one round trip instead
// of being silently merged via merge=union (which is advisory, not atomic).
const (
	TeamBranch       = "cavekit/team"
	TeamRef          = "refs/heads/" + TeamBranch
	TeamRemoteRef    = "refs/remotes/origin/" + TeamBranch
	LedgerFileName   = "ledger.jsonl"
	ledgerCommitName = "cavekit-team"
	ledgerCommitMail = "cavekit-team@local"
)

// RefClient owns all git-plumbing interaction with the team ledger ref. The
// Manager composes one of these and delegates append/fetch to it.
type RefClient struct {
	Root   string
	Exec   execpkg.Executor
	Stderr io.Writer
	Now    func() time.Time
}

func NewRefClient(root string, exec execpkg.Executor, stderr io.Writer) *RefClient {
	return &RefClient{
		Root:   root,
		Exec:   exec,
		Stderr: stderr,
		Now:    func() time.Time { return time.Now().UTC() },
	}
}

// HeadPath is a best-effort cache of the last observed ref sha so appends can
// CAS without an extra rev-parse.
func HeadPath(root string) string { return teamDirFile(root, "ledger.head") }

// OutboxPath holds events we tried to publish while offline. They are replayed
// on the next successful network operation.
func OutboxPath(root string) string { return teamDirFile(root, "outbox.jsonl") }

func teamDirFile(root, name string) string { return TeamDir(root) + "/" + name }

// EnsureRemoteBranch bootstraps the orphan branch on the origin. Safe to call
// repeatedly; a no-op if the ref already exists locally or on origin.
func (r *RefClient) EnsureRemoteBranch(ctx context.Context) error {
	// Try to fetch; if the ref exists remotely we're done.
	if err := r.Fetch(ctx); err == nil {
		if _, ok := r.remoteHead(ctx); ok {
			return nil
		}
	}
	// Ref does not exist yet — create an empty initial commit locally and push.
	seed := []LedgerEvent{}
	content, err := marshalLedger(seed)
	if err != nil {
		return err
	}
	// Migrate existing cache if present so early adopters don't lose history.
	if cached, err := os.ReadFile(LedgerPath(r.Root)); err == nil && len(cached) > 0 {
		content = cached
	}
	commit, err := r.buildCommit(ctx, content, "", "cavekit team: init ledger")
	if err != nil {
		return fmt.Errorf("seed ledger commit: %w", err)
	}
	if _, _, err := r.updateRef(ctx, TeamRef, commit, ""); err != nil {
		return fmt.Errorf("create local ledger ref: %w", err)
	}
	// Best-effort push. If offline, we'll retry on next publish.
	if _, _, err := r.pushCAS(ctx, commit, ""); err != nil {
		if r.Stderr != nil {
			fmt.Fprintf(r.Stderr, "warning: initial ledger push failed (will retry): %v\n", err)
		}
	} else {
		_, _, _ = r.updateRef(ctx, TeamRemoteRef, commit, "")
	}
	return r.writeHead(commit)
}

// Fetch pulls the latest ref from origin into refs/remotes/origin/cavekit/team.
// Returns nil even if the ref is missing on the remote; callers treat that as
// "no remote yet".
func (r *RefClient) Fetch(ctx context.Context) error {
	res, err := r.Exec.RunDir(ctx, r.Root, "git", "fetch", "origin",
		fmt.Sprintf("+%s:%s", TeamRef, TeamRemoteRef))
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		msg := strings.TrimSpace(res.Stderr)
		if isLocalOnlyPushFailure(msg) || strings.Contains(strings.ToLower(msg), "couldn't find remote ref") {
			return nil
		}
		return errors.New(msg)
	}
	return nil
}

// Read refreshes the local cache (.cavekit/team/ledger.jsonl) from the
// authoritative ref and returns the parsed events.
func (r *RefClient) Read(ctx context.Context, stderr io.Writer) ([]LedgerEvent, error) {
	// Prefer local ref if it exists (covers offline/newly-initialized setups);
	// else use the remote-tracking ref.
	sha, ok := r.localHead(ctx)
	if !ok {
		sha, ok = r.remoteHead(ctx)
	}
	if ok {
		if body, err := r.readBlob(ctx, sha); err == nil {
			if werr := os.WriteFile(LedgerPath(r.Root), body, 0o644); werr != nil {
				return nil, werr
			}
			_ = r.writeHead(sha)
		}
	}
	return ReadLedger(r.Root, stderr)
}

// Publish appends events to the ledger via CAS. On success the local cache and
// head are updated. On CAS rejection the caller should re-read, recheck
// invariants, and retry. On offline/network failure the events are queued to
// the outbox and the caller can choose to proceed (with a warning) or abort.
type PublishResult struct {
	CommitSHA  string
	Provisional bool // events landed in outbox, not yet on origin
	CASLost     bool
}

func (r *RefClient) Publish(ctx context.Context, events []LedgerEvent, message string, allowOffline bool) (PublishResult, error) {
	// Always drain any previous outbox first so that a successful publish flushes
	// accumulated history — even when the caller has no new events (used by
	// `team sync` as a catch-up hook).
	drained, _ := r.drainOutbox(ctx, allowOffline)
	events = append(drained, events...)
	if len(events) == 0 {
		return PublishResult{}, nil
	}

	// Try fetch to ensure we CAS against the latest head.
	_ = r.Fetch(ctx)

	var parent string
	if sha, ok := r.remoteHead(ctx); ok {
		parent = sha
	} else if sha, ok := r.localHead(ctx); ok {
		parent = sha
	}

	var body []byte
	if parent != "" {
		if b, err := r.readBlob(ctx, parent); err == nil {
			body = b
		}
	}
	body = appendEventLines(body, events)
	commit, err := r.buildCommit(ctx, body, parent, message)
	if err != nil {
		return PublishResult{}, err
	}

	stdout, stderr, err := r.pushCAS(ctx, commit, parent)
	if err == nil {
		_, _, _ = r.updateRef(ctx, TeamRemoteRef, commit, parent)
		_, _, _ = r.updateRef(ctx, TeamRef, commit, parent)
		_ = os.WriteFile(LedgerPath(r.Root), body, 0o644)
		_ = r.writeHead(commit)
		return PublishResult{CommitSHA: commit}, nil
	}

	// Classify the push failure. Use stderr only (not our synthetic err
	// message, which contains the word "rejected" unconditionally).
	lowerErr := strings.ToLower(stderr + "\n" + stdout)
	if isLocalOnlyPushFailure(stderr) {
		if !allowOffline {
			return PublishResult{}, &ExitError{
				Code:    7,
				Message: "team ledger push failed (offline); set allow_offline=true to queue provisionally: " + strings.TrimSpace(stderr),
			}
		}
		if qErr := r.enqueue(events); qErr != nil {
			return PublishResult{}, qErr
		}
		// Still update local cache so this session observes its own writes.
		_ = os.WriteFile(LedgerPath(r.Root), body, 0o644)
		return PublishResult{CommitSHA: commit, Provisional: true}, nil
	}
	if strings.Contains(lowerErr, "stale info") || strings.Contains(lowerErr, "rejected") || strings.Contains(lowerErr, "non-fast-forward") {
		return PublishResult{CASLost: true}, &ExitError{Code: 5, Message: "team ledger CAS lost; refetch and retry"}
	}
	return PublishResult{}, fmt.Errorf("team ledger push failed: %s", strings.TrimSpace(stderr))
}

// drainOutbox tries to publish queued offline events. If still offline it
// returns them so the caller can bundle them into the next attempt.
func (r *RefClient) drainOutbox(ctx context.Context, allowOffline bool) ([]LedgerEvent, error) {
	path := OutboxPath(r.Root)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var events []LedgerEvent
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var ev LedgerEvent
		if jerr := json.Unmarshal([]byte(line), &ev); jerr != nil {
			continue
		}
		events = append(events, ev)
	}
	// Caller re-publishes; clear the outbox now — if the retry fails we re-enqueue.
	_ = os.Remove(path)
	_ = allowOffline
	return events, nil
}

func (r *RefClient) enqueue(events []LedgerEvent) error {
	if err := ensureDir(TeamDir(r.Root)); err != nil {
		return err
	}
	f, err := os.OpenFile(OutboxPath(r.Root), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, ev := range events {
		data, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			return err
		}
	}
	return nil
}

// OutboxPendingCount exposes the queue depth for status reporting.
func OutboxPendingCount(root string) int {
	data, err := os.ReadFile(OutboxPath(root))
	if err != nil {
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// ----- git plumbing helpers --------------------------------------------------

func (r *RefClient) remoteHead(ctx context.Context) (string, bool) {
	res, err := r.Exec.RunDir(ctx, r.Root, "git", "rev-parse", "--verify", TeamRemoteRef)
	if err != nil || res.ExitCode != 0 {
		return "", false
	}
	sha := strings.TrimSpace(res.Stdout)
	return sha, sha != ""
}

func (r *RefClient) localHead(ctx context.Context) (string, bool) {
	res, err := r.Exec.RunDir(ctx, r.Root, "git", "rev-parse", "--verify", TeamRef)
	if err != nil || res.ExitCode != 0 {
		if sha := r.cachedHead(); sha != "" {
			return sha, true
		}
		return "", false
	}
	sha := strings.TrimSpace(res.Stdout)
	return sha, sha != ""
}

func (r *RefClient) cachedHead() string {
	data, err := os.ReadFile(HeadPath(r.Root))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (r *RefClient) writeHead(sha string) error {
	if err := ensureDir(TeamDir(r.Root)); err != nil {
		return err
	}
	return os.WriteFile(HeadPath(r.Root), []byte(sha+"\n"), 0o644)
}

// readBlob extracts ledger.jsonl from a commit's tree.
func (r *RefClient) readBlob(ctx context.Context, commitSHA string) ([]byte, error) {
	res, err := r.Exec.RunDir(ctx, r.Root, "git", "show", commitSHA+":"+LedgerFileName)
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, errors.New(strings.TrimSpace(res.Stderr))
	}
	return []byte(res.Stdout), nil
}

// buildCommit writes a blob, tree, and commit and returns the commit sha.
func (r *RefClient) buildCommit(ctx context.Context, body []byte, parent, message string) (string, error) {
	// Write the blob via stdin. The MockExecutor doesn't pipe stdin, so for
	// testing we instead stage the body in a temp file and use `git hash-object -w --path`.
	tmp, err := os.CreateTemp(TeamDir(r.Root), "ledger-*.jsonl")
	if err != nil {
		if err := ensureDir(TeamDir(r.Root)); err != nil {
			return "", err
		}
		tmp, err = os.CreateTemp(TeamDir(r.Root), "ledger-*.jsonl")
		if err != nil {
			return "", err
		}
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return "", err
	}
	tmp.Close()

	blobRes, err := r.Exec.RunDir(ctx, r.Root, "git", "hash-object", "-w", tmpPath)
	if err != nil {
		return "", err
	}
	if blobRes.ExitCode != 0 {
		return "", errors.New(strings.TrimSpace(blobRes.Stderr))
	}
	blob := strings.TrimSpace(blobRes.Stdout)

	treeInput := fmt.Sprintf("100644 blob %s\t%s\n", blob, LedgerFileName)
	treeRes, err := r.runWithStdin(ctx, treeInput, "git", "mktree")
	if err != nil {
		return "", err
	}
	if treeRes.ExitCode != 0 {
		return "", errors.New(strings.TrimSpace(treeRes.Stderr))
	}
	tree := strings.TrimSpace(treeRes.Stdout)

	args := []string{"commit-tree", tree, "-m", message}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	// Pin author/committer so cross-device commits are deterministic and don't
	// depend on the dev's local git config.
	env := map[string]string{
		"GIT_AUTHOR_NAME":     ledgerCommitName,
		"GIT_AUTHOR_EMAIL":    ledgerCommitMail,
		"GIT_COMMITTER_NAME":  ledgerCommitName,
		"GIT_COMMITTER_EMAIL": ledgerCommitMail,
	}
	commitRes, err := r.runWithEnv(ctx, env, "git", args...)
	if err != nil {
		return "", err
	}
	if commitRes.ExitCode != 0 {
		return "", errors.New(strings.TrimSpace(commitRes.Stderr))
	}
	return strings.TrimSpace(commitRes.Stdout), nil
}

// pushCAS pushes commit to the team branch using --force-with-lease so the
// push is rejected if another client advanced the ref since we last read it.
func (r *RefClient) pushCAS(ctx context.Context, commit, expected string) (string, string, error) {
	lease := TeamRef
	if expected != "" {
		lease = TeamRef + ":" + expected
	}
	args := []string{"push", "origin", commit + ":" + TeamRef, "--force-with-lease=" + lease}
	res, err := r.Exec.RunDir(ctx, r.Root, "git", args...)
	if err != nil {
		return "", "", err
	}
	if res.ExitCode != 0 {
		return res.Stdout, res.Stderr, fmt.Errorf("push rejected (exit %d)", res.ExitCode)
	}
	return res.Stdout, res.Stderr, nil
}

func (r *RefClient) updateRef(ctx context.Context, ref, newSHA, oldSHA string) (string, string, error) {
	args := []string{"update-ref", ref, newSHA}
	if oldSHA != "" {
		args = append(args, oldSHA)
	}
	res, err := r.Exec.RunDir(ctx, r.Root, "git", args...)
	if err != nil {
		return "", "", err
	}
	if res.ExitCode != 0 {
		return res.Stdout, res.Stderr, fmt.Errorf("update-ref failed: %s", strings.TrimSpace(res.Stderr))
	}
	return res.Stdout, res.Stderr, nil
}

// runWithStdin/runWithEnv wrap the Executor to carry stdin and environment for
// plumbing calls. MockExecutor ignores these, which is fine — the mock handlers
// short-circuit these commands with synthesized output.
func (r *RefClient) runWithStdin(ctx context.Context, stdin, name string, args ...string) (execpkg.Result, error) {
	if se, ok := r.Exec.(StdinExecutor); ok {
		return se.RunDirStdin(ctx, r.Root, stdin, name, args...)
	}
	return r.Exec.RunDir(ctx, r.Root, name, args...)
}

func (r *RefClient) runWithEnv(ctx context.Context, env map[string]string, name string, args ...string) (execpkg.Result, error) {
	if ee, ok := r.Exec.(EnvExecutor); ok {
		return ee.RunDirEnv(ctx, r.Root, env, name, args...)
	}
	return r.Exec.RunDir(ctx, r.Root, name, args...)
}

// Optional capabilities on the executor. Real executor implements both; mocks
// can skip them and handlers will see the raw command only.
type StdinExecutor interface {
	RunDirStdin(ctx context.Context, dir, stdin, name string, args ...string) (execpkg.Result, error)
}
type EnvExecutor interface {
	RunDirEnv(ctx context.Context, dir string, env map[string]string, name string, args ...string) (execpkg.Result, error)
}

// appendEventLines serializes events and appends them to an existing JSONL body.
func appendEventLines(existing []byte, events []LedgerEvent) []byte {
	out := append([]byte{}, existing...)
	if len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	for _, ev := range events {
		data, err := json.Marshal(ev)
		if err != nil {
			continue
		}
		out = append(out, data...)
		out = append(out, '\n')
	}
	return out
}

func marshalLedger(events []LedgerEvent) ([]byte, error) {
	if len(events) == 0 {
		return []byte{}, nil
	}
	return appendEventLines(nil, events), nil
}
