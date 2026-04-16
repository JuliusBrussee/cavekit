package main

import (
	"errors"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
)

func runSetupBuild(args []string) {
	sourceDir, remaining, err := parseSourceDirFlag(args)
	exitOnError(err)

	if runtime.GOOS == "windows" {
		exitOnError(errors.New("cavekit setup-build is not ported natively on Windows yet"))
	}

	scriptPath := filepath.Join(resolveSourceDir(sourceDir), "scripts", "setup-build.sh")
	cmd := osexec.Command(scriptPath, remaining...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	exitOnError(cmd.Run())
}
