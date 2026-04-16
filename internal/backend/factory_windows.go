//go:build windows

package backend

import (
	"github.com/JuliusBrussee/cavekit/internal/exec"
	"github.com/JuliusBrussee/cavekit/internal/windowspty"
)

func New(executor exec.Executor) (SessionBackend, Capabilities) {
	return windowspty.NewManager(executor), Capabilities{
		SupportsAttach:        false,
		SupportsShellTerminal: false,
	}
}
