package team

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	execpkg "github.com/JuliusBrussee/cavekit/internal/exec"
)

const (
	hookMarker = "# cavekit-team-guard"
	hookBody   = `#!/bin/sh
# cavekit-team-guard
# Rejects commits that touch files claimed by a teammate. Bypass with
# CAVEKIT_TEAM_OVERRIDE=1 (records an override event in the ledger).
if [ -n "$CAVEKIT_TEAM_SKIP_GUARD" ]; then
  exit 0
fi
exec cavekit team guard-commit
`
)

// InstallCommitHook writes .git/hooks/pre-commit that invokes `cavekit team
// guard-commit`. If an unrelated pre-commit hook already exists it is left
// untouched unless --force; instead we append a managed snippet.
func InstallCommitHook(root string, force bool) error {
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := ensureDir(hooksDir); err != nil {
		return err
	}
	path := filepath.Join(hooksDir, "pre-commit")
	if data, err := os.ReadFile(path); err == nil {
		if strings.Contains(string(data), hookMarker) {
			return nil
		}
		if !force {
			// Append our invocation guarded by the marker so a custom hook can coexist.
			appended := strings.TrimRight(string(data), "\n") + "\n\n" + hookMarker + "\nif [ -z \"$CAVEKIT_TEAM_SKIP_GUARD\" ]; then cavekit team guard-commit || exit 1; fi\n"
			return os.WriteFile(path, []byte(appended), 0o755)
		}
	}
	if err := os.WriteFile(path, []byte(hookBody), 0o755); err != nil {
		return err
	}
	return nil
}

// GuardCommit is invoked as the pre-commit hook. It resolves the staged file
// list, reads the ledger, and rejects the commit if any staged path is owned
// by another active session. Override via CAVEKIT_TEAM_OVERRIDE=1 (recorded as
// a note event).
func GuardCommit(ctx context.Context, root string, exec execpkg.Executor, stderr io.Writer) error {
	// Bail silently when team isn't initialized — no enforcement until opt-in.
	if !IsInitialized(root) || !fileExists(IdentityPath(root)) {
		return nil
	}

	staged, err := listStagedPaths(ctx, root, exec)
	if err != nil {
		return err
	}
	if len(staged) == 0 {
		return nil
	}

	identity, err := ReadIdentity(root)
	if err != nil {
		return err
	}
	events, err := ReadLedger(root, stderr)
	if err != nil {
		return err
	}
	cfg, err := LoadConfig(root)
	if err != nil {
		return err
	}

	claims := AllActiveClaims(events, time.Duration(cfg.LeaseTTLSeconds)*time.Second, time.Now().UTC())
	var violations []string
	for _, path := range staged {
		for _, claim := range claims {
			if claim.Session == identity.Session || claim.Owner == identity.Email {
				continue
			}
			if len(claim.Paths) == 0 {
				// Unscoped claims reserve nothing in particular; ignore for guard purposes.
				continue
			}
			if MatchAny(claim.Paths, path) {
				violations = append(violations,
					fmt.Sprintf("%s → claimed by %s on %s (paths: %s)",
						path, claim.Owner, claim.Task, strings.Join(claim.Paths, ", ")))
				break
			}
		}
	}

	if len(violations) == 0 {
		return nil
	}

	if os.Getenv("CAVEKIT_TEAM_OVERRIDE") == "1" {
		_ = AppendLedgerEvent(root, LedgerEvent{
			TS:      time.Now().UTC().Format(time.RFC3339),
			Type:    EventNote,
			Owner:   identity.Email,
			Session: identity.Session,
			Note:    "commit-guard override: " + strings.Join(violations, "; "),
		})
		return nil
	}

	if stderr != nil {
		fmt.Fprintln(stderr, "cavekit team: commit blocked — staged files are claimed by teammates:")
		for _, v := range violations {
			fmt.Fprintln(stderr, "  "+v)
		}
		fmt.Fprintln(stderr, "Options:")
		fmt.Fprintln(stderr, "  1. Coordinate — ask them to release the task.")
		fmt.Fprintln(stderr, "  2. Switch tasks via `cavekit team next`.")
		fmt.Fprintln(stderr, "  3. Emergency override: CAVEKIT_TEAM_OVERRIDE=1 git commit ...")
	}
	return &ExitError{Code: 8, Message: "commit blocked by cavekit team guard"}
}

// listStagedPaths returns the paths currently staged for commit, normalized
// to forward slashes.
func listStagedPaths(ctx context.Context, root string, exec execpkg.Executor) ([]string, error) {
	res, err := exec.RunDir(ctx, root, "git", "diff", "--cached", "--name-only", "--diff-filter=ACMRT")
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("git diff --cached: %s", strings.TrimSpace(res.Stderr))
	}
	var paths []string
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		paths = append(paths, filepath.ToSlash(line))
	}
	return paths, nil
}
