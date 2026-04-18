package team

import (
	"context"
	"testing"

	execpkg "github.com/JuliusBrussee/cavekit/internal/exec"
)

func TestResolveIdentity_UsesFlagPrecedence(t *testing.T) {
	mock := execpkg.NewMockExecutor()
	mock.OnCommand("git", func(call execpkg.Call) (execpkg.Result, error) {
		return execpkg.Result{Stdout: "repo@example.com\n", ExitCode: 0}, nil
	})

	identity, err := ResolveIdentity(context.Background(), mock, t.TempDir(), "flag@example.com", "Flag User")
	if err != nil {
		t.Fatalf("resolve identity: %v", err)
	}
	if identity.Email != "flag@example.com" {
		t.Fatalf("expected flag email, got %s", identity.Email)
	}
	if identity.Name != "Flag User" {
		t.Fatalf("expected flag name, got %s", identity.Name)
	}
	if identity.Session == "" {
		t.Fatal("expected session id")
	}
}

func TestResolveIdentity_FailsWithoutGitEmail(t *testing.T) {
	mock := execpkg.NewMockExecutor()
	mock.DefaultResult = execpkg.Result{ExitCode: 1}

	_, err := ResolveIdentity(context.Background(), mock, t.TempDir(), "", "")
	if err == nil {
		t.Fatal("expected an error")
	}
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.Code != 2 {
		t.Fatalf("expected exit code 2, got %d", exitErr.Code)
	}
}
