package exec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	sep := -1
	for i, arg := range args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if sep == -1 || sep+1 >= len(args) {
		os.Exit(2)
	}

	switch args[sep+1] {
	case "echo":
		if sep+2 < len(args) {
			fmt.Fprintln(os.Stdout, args[sep+2])
		}
		os.Exit(0)
	case "exit42":
		os.Exit(42)
	case "pwd":
		wd, err := os.Getwd()
		if err != nil {
			os.Exit(2)
		}
		fmt.Fprintln(os.Stdout, wd)
		os.Exit(0)
	default:
		os.Exit(2)
	}
}

func TestRealExecutor_Echo(t *testing.T) {
	e := NewRealExecutor()
	name, args, env := helperCommand("echo", "hello")
	withEnv(t, env, func() {
		res, err := e.Run(context.Background(), name, args...)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Stdout != "hello\n" {
			t.Errorf("got stdout=%q, want %q", res.Stdout, "hello\n")
		}
		if res.ExitCode != 0 {
			t.Errorf("got exit code %d, want 0", res.ExitCode)
		}
	})
}

func TestRealExecutor_NonZeroExit(t *testing.T) {
	e := NewRealExecutor()
	name, args, env := helperCommand("exit42")
	withEnv(t, env, func() {
		res, err := e.Run(context.Background(), name, args...)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ExitCode != 42 {
			t.Errorf("got exit code %d, want 42", res.ExitCode)
		}
	})
}

func TestRealExecutor_RunDir(t *testing.T) {
	e := NewRealExecutor()
	tmpDir := t.TempDir()
	name, args, env := helperCommand("pwd")
	withEnv(t, env, func() {
		res, err := e.RunDir(context.Background(), tmpDir, name, args...)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := filepath.Clean(stringsTrimSpace(res.Stdout))
		if got != filepath.Clean(tmpDir) {
			t.Fatalf("got stdout=%q, want %q", got, tmpDir)
		}
	})
}

func helperCommand(args ...string) (string, []string, []string) {
	cmdArgs := []string{"-test.run=TestHelperProcess", "--"}
	cmdArgs = append(cmdArgs, args...)
	return os.Args[0], cmdArgs, []string{"GO_WANT_HELPER_PROCESS=1"}
}

func withEnv(t *testing.T, env []string, fn func()) {
	t.Helper()
	previous := make([]string, 0, len(env))
	for _, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		key := parts[0]
		previous = append(previous, key+"="+os.Getenv(key))
		if len(parts) == 2 {
			if err := os.Setenv(key, parts[1]); err != nil {
				t.Fatalf("setenv %s: %v", key, err)
			}
		}
	}
	defer func() {
		for _, entry := range previous {
			parts := strings.SplitN(entry, "=", 2)
			key := parts[0]
			val := ""
			if len(parts) == 2 {
				val = parts[1]
			}
			if val == "" {
				_ = os.Unsetenv(key)
				continue
			}
			_ = os.Setenv(key, val)
		}
	}()

	fn()
}

func stringsTrimSpace(s string) string {
	return strings.TrimSpace(s)
}

func TestMockExecutor_RecordsCalls(t *testing.T) {
	m := NewMockExecutor()
	m.DefaultResult = Result{Stdout: "ok\n"}

	res, err := m.Run(context.Background(), "tmux", "has-session", "-t", "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Stdout != "ok\n" {
		t.Errorf("got %q, want %q", res.Stdout, "ok\n")
	}
	if len(m.Calls) != 1 {
		t.Fatalf("got %d calls, want 1", len(m.Calls))
	}
	if m.Calls[0].Name != "tmux" {
		t.Errorf("got name=%q, want %q", m.Calls[0].Name, "tmux")
	}
}

func TestMockExecutor_Handler(t *testing.T) {
	m := NewMockExecutor()
	m.OnCommand("git", func(c Call) (Result, error) {
		return Result{Stdout: "main\n"}, nil
	})

	res, _ := m.Run(context.Background(), "git", "branch", "--show-current")
	if res.Stdout != "main\n" {
		t.Errorf("got %q, want %q", res.Stdout, "main\n")
	}
}
