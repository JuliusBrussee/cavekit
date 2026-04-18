package team

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	execpkg "github.com/JuliusBrussee/cavekit/internal/exec"
)

// fakeExec simulates a git CLI for Manager tests. It knows enough git plumbing
// surface to drive Publish() through RefClient without a real repository.
type fakeExec struct {
	calls      []execpkg.Call
	handler    func(execpkg.Call) (execpkg.Result, error)
	blobs      map[string][]byte // sha → blob bytes (for git show commit:file)
	commits    map[string]string // commit sha → its ledger blob sha
	nextBlob   int
	nextCommit int
	pushMode   string // "offline", "success", "reject"
}

func newFakeExec() *fakeExec {
	return &fakeExec{
		blobs:   map[string][]byte{},
		commits: map[string]string{},
	}
}

func (f *fakeExec) Run(ctx context.Context, name string, args ...string) (execpkg.Result, error) {
	return f.RunDir(ctx, "", name, args...)
}

func (f *fakeExec) RunDir(ctx context.Context, dir, name string, args ...string) (execpkg.Result, error) {
	return f.dispatch(execpkg.Call{Dir: dir, Name: name, Args: args}, "")
}

func (f *fakeExec) RunDirStdin(ctx context.Context, dir, stdin, name string, args ...string) (execpkg.Result, error) {
	return f.dispatch(execpkg.Call{Dir: dir, Name: name, Args: args}, stdin)
}

func (f *fakeExec) RunDirEnv(ctx context.Context, dir string, env map[string]string, name string, args ...string) (execpkg.Result, error) {
	return f.dispatch(execpkg.Call{Dir: dir, Name: name, Args: args}, "")
}

func (f *fakeExec) dispatch(call execpkg.Call, stdin string) (execpkg.Result, error) {
	f.calls = append(f.calls, call)
	if f.handler != nil {
		return f.handler(call)
	}
	if call.Name != "git" || len(call.Args) == 0 {
		return execpkg.Result{}, nil
	}
	sub := call.Args[0]
	switch sub {
	case "fetch":
		return execpkg.Result{ExitCode: 0}, nil
	case "rev-parse":
		// --verify refs/...: reply "not found" so RefClient treats as uninitialized.
		return execpkg.Result{ExitCode: 1, Stderr: "fatal: unknown revision"}, nil
	case "hash-object":
		sha := f.allocBlob(stdin)
		// Our buildCommit writes body to a tmp file; --path is on it. Read file back.
		for _, a := range call.Args {
			if strings.HasPrefix(a, "/") || strings.Contains(a, ".cavekit") {
				if data, err := os.ReadFile(a); err == nil {
					sha = f.allocBlob(string(data))
				}
			}
		}
		return execpkg.Result{ExitCode: 0, Stdout: sha + "\n"}, nil
	case "mktree":
		// Stub tree: return a synthetic sha.
		return execpkg.Result{ExitCode: 0, Stdout: f.allocTree() + "\n"}, nil
	case "commit-tree":
		commit := f.allocCommit()
		// Link commit → latest blob we created so `git show commit:file` works.
		if last := f.lastBlob(); last != "" {
			f.commits[commit] = last
		}
		return execpkg.Result{ExitCode: 0, Stdout: commit + "\n"}, nil
	case "update-ref":
		return execpkg.Result{ExitCode: 0}, nil
	case "show":
		// "<commit>:ledger.jsonl"
		if len(call.Args) >= 2 {
			parts := strings.SplitN(call.Args[1], ":", 2)
			if len(parts) == 2 {
				if blobSha, ok := f.commits[parts[0]]; ok {
					return execpkg.Result{ExitCode: 0, Stdout: string(f.blobs[blobSha])}, nil
				}
			}
		}
		return execpkg.Result{ExitCode: 0, Stdout: ""}, nil
	case "push":
		switch f.pushMode {
		case "offline":
			return execpkg.Result{ExitCode: 128, Stderr: "fatal: no configured push destination"}, nil
		case "reject":
			return execpkg.Result{ExitCode: 1, Stderr: "! [rejected] stale info"}, nil
		default:
			return execpkg.Result{ExitCode: 0}, nil
		}
	}
	return execpkg.Result{ExitCode: 0}, nil
}

func (f *fakeExec) allocBlob(body string) string {
	f.nextBlob++
	sha := fakeSHA("blob", f.nextBlob)
	f.blobs[sha] = []byte(body)
	return sha
}

func (f *fakeExec) allocTree() string {
	f.nextBlob++
	return fakeSHA("tree", f.nextBlob)
}

func (f *fakeExec) allocCommit() string {
	f.nextCommit++
	return fakeSHA("commit", f.nextCommit)
}

func (f *fakeExec) lastBlob() string {
	var last string
	// Deterministic: pick whichever has max counter suffix.
	for sha := range f.blobs {
		if sha > last {
			last = sha
		}
	}
	return last
}

func fakeSHA(prefix string, n int) string {
	s := prefix + "-" + padHex(n)
	// Pad to 40 chars so strings.TrimSpace comparisons remain stable.
	if len(s) < 40 {
		s = s + strings.Repeat("0", 40-len(s))
	}
	return s[:40]
}

func padHex(n int) string {
	const hex = "0123456789abcdef"
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{hex[n%16]}, out...)
		n /= 16
	}
	return string(out)
}

// seedProject writes the minimum project layout for Manager.Claim to work.
func seedProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "context", "plans"), 0o755); err != nil {
		t.Fatalf("mkdir plans: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "context", "impl"), 0o755); err != nil {
		t.Fatalf("mkdir impl: %v", err)
	}
	plan := `## Tier 1
| Task | Title | Spec | Requirement | blockedBy | Effort |
| T-001 | Demo task | demo | R1 | — | S |
`
	if err := os.WriteFile(filepath.Join(root, "context", "plans", "plan-site.md"), []byte(plan), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	if err := EnsureLedger(root); err != nil {
		t.Fatalf("ensure ledger: %v", err)
	}
	if err := WriteDefaultConfig(root); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := WriteIdentity(root, Identity{
		Email:    "alice@example.com",
		Session:  "session-alice",
		JoinedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("write identity: %v", err)
	}
	return root
}

func newTestManager(root string, ex execpkg.Executor, stderr io.Writer) *Manager {
	m := NewManager(root, ex, stderr)
	m.Hostname = "test-host"
	m.PID = 42
	return m
}

// TestClaim_OfflineStrict: when push fails with a local-only error and
// AllowOffline=false, claim must fail and clean up.
func TestClaim_OfflineStrict(t *testing.T) {
	root := seedProject(t)
	ex := newFakeExec()
	ex.pushMode = "offline"

	var stderr bytes.Buffer
	m := newTestManager(root, ex, &stderr)

	_, err := m.Claim(context.Background(), "T-001", nil)
	if err == nil {
		t.Fatal("expected claim to fail offline")
	}
	exitErr, ok := err.(*ExitError)
	if !ok || exitErr.Code != 7 {
		t.Fatalf("expected exit 7, got %v", err)
	}
	if _, statErr := os.Stat(LeasePath(root, "T-001")); !os.IsNotExist(statErr) {
		t.Fatalf("lease should have been cleaned up; stat err=%v", statErr)
	}
	if OutboxPendingCount(root) != 0 {
		t.Fatalf("strict mode must not queue to outbox")
	}
}

// TestClaim_OfflineAllowedQueuesProvisional: AllowOffline=true permits the
// claim, marks it provisional, and writes to outbox for later replay.
func TestClaim_OfflineAllowedQueuesProvisional(t *testing.T) {
	root := seedProject(t)

	cfg := DefaultConfig()
	cfg.AllowOffline = true
	if err := writeJSON(ConfigPath(root), cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}

	ex := newFakeExec()
	ex.pushMode = "offline"

	var stderr bytes.Buffer
	m := newTestManager(root, ex, &stderr)

	res, err := m.Claim(context.Background(), "T-001", []string{"src/auth/**"})
	if err != nil {
		t.Fatalf("expected offline-allowed claim to succeed, got %v", err)
	}
	if !res.Provisional {
		t.Fatal("expected claim to be marked provisional")
	}
	if OutboxPendingCount(root) == 0 {
		t.Fatal("expected outbox to queue the provisional event")
	}
}

// TestClaim_SucceedsOnline drives the full CAS publish through the fake git
// executor with a successful push and asserts the ledger shows the claim.
func TestClaim_SucceedsOnline(t *testing.T) {
	root := seedProject(t)
	ex := newFakeExec()

	var stderr bytes.Buffer
	m := newTestManager(root, ex, &stderr)

	res, err := m.Claim(context.Background(), "T-001", []string{"src/demo/**"})
	if err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	if res.Provisional {
		t.Fatal("online claim should not be provisional")
	}
	if res.CommitSHA == "" {
		t.Fatal("expected a commit sha from Publish")
	}
	// Local cache should record the event.
	events, err := ReadLedger(root, nil)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Task == "T-001" && e.Type == EventClaim {
			if len(e.Paths) == 0 {
				t.Fatalf("expected paths recorded on claim event")
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("claim event missing from ledger")
	}
}

// TestPathsOverlap sanity-checks the glob-overlap detector so the scheduler
// and guard don't regress silently.
func TestPathsOverlap(t *testing.T) {
	cases := []struct {
		a, b []string
		want bool
		name string
	}{
		{[]string{"src/auth/**"}, []string{"src/auth/login.go"}, true, "prefix vs file"},
		{[]string{"src/auth/**"}, []string{"src/billing/**"}, false, "disjoint trees"},
		{[]string{"**/*.go"}, []string{"src/auth/login.go"}, true, "double-star file"},
		{[]string{}, []string{"x"}, true, "empty is unscoped"},
		{[]string{"src/"}, []string{"src/foo.go"}, true, "directory prefix"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := PathsOverlap(c.a, c.b); got != c.want {
				t.Fatalf("PathsOverlap(%v,%v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

// TestGuardCommit_BlocksTeammatePaths writes a ledger with another session's
// active claim covering src/auth/** and asserts GuardCommit exits 8 when the
// local identity stages src/auth/login.go.
func TestGuardCommit_BlocksTeammatePaths(t *testing.T) {
	root := seedProject(t)
	// Append a rival claim to the ledger cache.
	now := time.Now().UTC()
	rival := LedgerEvent{
		TS:         now.Format(time.RFC3339),
		Type:       EventClaim,
		Task:       "T-002",
		Owner:      "bob@example.com",
		Session:    "session-bob",
		LeaseUntil: now.Add(30 * time.Minute).Format(time.RFC3339),
		Paths:      []string{"src/auth/**"},
	}
	if err := AppendLedgerEvent(root, rival); err != nil {
		t.Fatalf("append rival: %v", err)
	}

	ex := newFakeExec()
	ex.handler = func(call execpkg.Call) (execpkg.Result, error) {
		if call.Name == "git" && len(call.Args) >= 2 && call.Args[0] == "diff" {
			return execpkg.Result{ExitCode: 0, Stdout: "src/auth/login.go\n"}, nil
		}
		return execpkg.Result{ExitCode: 0}, nil
	}

	err := GuardCommit(context.Background(), root, ex, io.Discard)
	if err == nil {
		t.Fatal("expected guard to block commit")
	}
	if exitErr, ok := err.(*ExitError); !ok || exitErr.Code != 8 {
		t.Fatalf("expected exit 8, got %v", err)
	}
}
