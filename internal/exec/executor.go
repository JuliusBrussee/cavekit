// Package exec provides a command executor abstraction for testability.
// Production code uses RealExecutor which shells out to real commands.
// Tests can substitute a mock that records and replays command outputs.
package exec

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
)

// Result holds the output of a command execution.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Executor abstracts command execution for testability.
type Executor interface {
	// Run executes a command and returns its result.
	Run(ctx context.Context, name string, args ...string) (Result, error)
	// RunDir executes a command in a specific working directory.
	RunDir(ctx context.Context, dir string, name string, args ...string) (Result, error)
}

// RealExecutor shells out to actual system commands.
type RealExecutor struct{}

func NewRealExecutor() *RealExecutor {
	return &RealExecutor{}
}

func (e *RealExecutor) Run(ctx context.Context, name string, args ...string) (Result, error) {
	return e.RunDir(ctx, "", name, args...)
}

func (e *RealExecutor) RunDir(ctx context.Context, dir string, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
		err = nil // non-zero exit is not an execution error
	}

	return Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, err
}

// RunDirStdin is like RunDir but pipes stdin into the command.
func (e *RealExecutor) RunDirStdin(ctx context.Context, dir, stdin, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
		err = nil
	}
	return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitCode}, err
}

// RunDirEnv is like RunDir but extends the child's environment with the given
// pairs. Used by plumbing calls that need deterministic author/committer.
func (e *RealExecutor) RunDirEnv(ctx context.Context, dir string, env map[string]string, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	parentEnv := os.Environ()
	for k, v := range env {
		parentEnv = append(parentEnv, k+"="+v)
	}
	cmd.Env = parentEnv
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
		err = nil
	}
	return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitCode}, err
}
