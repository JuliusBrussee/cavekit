package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	execpkg "github.com/JuliusBrussee/cavekit/internal/exec"
	"github.com/JuliusBrussee/cavekit/internal/worktree"
)

func effectiveHomeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if home := os.Getenv("USERPROFILE"); home != "" {
		return home
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

func effectiveLocalAppData() string {
	if dir := os.Getenv("CAVEKIT_LOCALAPPDATA"); dir != "" {
		return dir
	}
	if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
		return dir
	}
	return filepath.Join(effectiveHomeDir(), "AppData", "Local")
}

func currentProjectRootOrCwd() string {
	if root := os.Getenv("BP_PROJECT_ROOT"); root != "" {
		return filepath.Clean(root)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}

	executor := execpkg.NewRealExecutor()
	wtMgr := worktree.NewManager(executor)
	root, err := wtMgr.ProjectRoot(context.Background(), cwd)
	if err != nil {
		return cwd
	}
	return root
}

func parseSourceDirFlag(args []string) (string, []string, error) {
	var sourceDir string
	remaining := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--source-dir":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--source-dir requires a value")
			}
			sourceDir = args[i+1]
			i++
		case strings.HasPrefix(arg, "--source-dir="):
			sourceDir = strings.TrimPrefix(arg, "--source-dir=")
		default:
			remaining = append(remaining, arg)
		}
	}

	return sourceDir, remaining, nil
}

func resolveSourceDir(explicit string) string {
	if explicit != "" {
		return filepath.Clean(explicit)
	}

	root := currentProjectRootOrCwd()
	if dirExists(filepath.Join(root, "commands")) {
		return root
	}

	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		if dirExists(filepath.Join(exeDir, "commands")) {
			return exeDir
		}
		parent := filepath.Dir(exeDir)
		if dirExists(filepath.Join(parent, "commands")) {
			return parent
		}
	}

	return root
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := ensureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func removeIfExists(path string) error {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.RemoveAll(path)
}

func writeJSONFile(path string, value any) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func exitOnError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
