//go:build !windows

package backend

import (
	"github.com/JuliusBrussee/cavekit/internal/exec"
	"github.com/JuliusBrussee/cavekit/internal/tmux"
)

func New(executor exec.Executor) (SessionBackend, Capabilities) {
	return tmux.NewManager(executor), Capabilities{
		SupportsAttach:        true,
		SupportsShellTerminal: true,
	}
}
