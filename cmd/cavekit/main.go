package main

import (
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"

	"github.com/JuliusBrussee/cavekit/internal/exec"
	"github.com/JuliusBrussee/cavekit/internal/session"
	"github.com/JuliusBrussee/cavekit/internal/site"
	"github.com/JuliusBrussee/cavekit/internal/tmux"
	"github.com/JuliusBrussee/cavekit/internal/tui"
	"github.com/JuliusBrussee/cavekit/internal/worktree"
)

const version = "v0.1.0"

func main() {
	cmd := "monitor"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "monitor", "":
		runMonitor()
	case "status":
		runStatus()
	case "team":
		runTeam()
	case "kill":
		runKill()
	case "version":
		fmt.Println("cavekit", version)
	case "debug":
		runDebug()
	case "reset":
		runReset()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		fmt.Fprintln(os.Stderr, "usage: cavekit [monitor|status|team|kill|version|debug|reset]")
		os.Exit(1)
	}
}

func runMonitor() {
	// Parse flags
	program := "claude"
	autoYes := false
	for i, arg := range os.Args {
		if (arg == "--program" || arg == "-p") && i+1 < len(os.Args) {
			program = os.Args[i+1]
		}
		if arg == "--autoyes" || arg == "-y" {
			autoYes = true
		}
	}

	// Preflight checks
	if err := preflight(program); err != nil {
		fmt.Fprintf(os.Stderr, "preflight failed: %s\n", err)
		os.Exit(1)
	}

	// Determine project root
	cwd, _ := os.Getwd()
	executor := exec.NewRealExecutor()
	wtMgr := worktree.NewManager(executor)
	ctx := context.Background()
	root, err := wtMgr.ProjectRoot(ctx, cwd)
	if err != nil {
		root = cwd
	}

	// Launch TUI
	if err := tui.Run(root, program, autoYes); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %s\n", err)
		os.Exit(1)
	}
}

func runStatus() {
	executor := exec.NewRealExecutor()
	wtMgr := worktree.NewManager(executor)
	ctx := context.Background()

	cwd, _ := os.Getwd()
	root, err := wtMgr.ProjectRoot(ctx, cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "not in a git repo: %s\n", err)
		os.Exit(1)
	}

	// Autonomous runtime status first, when present.
	if st, ok := readCavekitStatus(root); ok {
		fmt.Println(st)
		fmt.Println()
	}

	worktrees, err := worktree.DiscoverAll(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "discover worktrees: %s\n", err)
		os.Exit(1)
	}

	if len(worktrees) == 0 {
		fmt.Println("No Cavekit worktrees found.")
		return
	}

	for _, wt := range worktrees {
		icon := "·"
		if wt.HasRalphLoop {
			icon = "⟳"
		}

		// Try to compute progress
		done, total := computeWorktreeProgress(wt.Path)
		if total > 0 {
			fmt.Printf("%s %s: %d/%d tasks done\n", icon, wt.SiteName, done, total)
		} else {
			fmt.Printf("%s %s: %s\n", icon, wt.SiteName, wt.Path)
		}
	}
}

// readCavekitStatus runs cavekit-tools.cjs status in the project root when a
// runtime state file exists, returning the printed status block.
func readCavekitStatus(root string) (string, bool) {
	stateFile := filepath.Join(root, ".cavekit", "state.md")
	if _, err := os.Stat(stateFile); err != nil {
		return "", false
	}
	node, err := osexec.LookPath("node")
	if err != nil {
		return "", false
	}
	// Resolve plugin script path relative to this binary. Prefer an env override.
	script := os.Getenv("CAVEKIT_TOOLS_SCRIPT")
	if script == "" {
		// Look for scripts/cavekit-tools.cjs next to the binary or the repo root.
		candidates := []string{
			filepath.Join(root, "scripts", "cavekit-tools.cjs"),
			filepath.Join(os.Getenv("CLAUDE_PLUGIN_ROOT"), "scripts", "cavekit-tools.cjs"),
		}
		for _, c := range candidates {
			if c == "" {
				continue
			}
			if _, err := os.Stat(c); err == nil {
				script = c
				break
			}
		}
	}
	if script == "" {
		return "", false
	}
	cmd := osexec.Command(node, script, "status", "--cavekit-dir", filepath.Join(root, ".cavekit"))
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

// computeWorktreeProgress reads site and impl files to compute task progress.
func computeWorktreeProgress(wtPath string) (done, total int) {
	// Look for site files in worktree
	sitesDir := filepath.Join(wtPath, "context", "sites")
	sites, err := site.Discover(sitesDir)
	if err != nil || len(sites) == 0 {
		return 0, 0
	}

	// Parse first site
	f, err := site.Parse(sites[0].Path)
	if err != nil {
		return 0, 0
	}

	// Track status from impl files
	implDir := filepath.Join(wtPath, "context", "impl")
	statuses, err := site.TrackStatus(implDir)
	if err != nil {
		return 0, len(f.Tasks)
	}

	summary := site.ComputeProgress(f, statuses)
	return summary.Done, summary.Total
}

func runKill() {
	executor := exec.NewRealExecutor()
	tmuxMgr := tmux.NewManager(executor)
	wtMgr := worktree.NewManager(executor)
	ctx := context.Background()

	cwd, _ := os.Getwd()
	root, _ := wtMgr.ProjectRoot(ctx, cwd)

	// Kill tmux sessions
	sessions, _ := tmuxMgr.ListSessions(ctx)
	killed := 0
	for _, s := range sessions {
		tmuxMgr.Kill(ctx, s)
		killed++
	}

	// Remove worktrees and branches
	worktrees, _ := worktree.DiscoverAll(root)
	cleaned := 0
	for _, wt := range worktrees {
		wtMgr.Remove(ctx, root, wt.SiteName)
		cleaned++
	}

	// Clear persisted state
	store := session.NewStore("")
	os.Remove(store.Path())

	fmt.Printf("Killed %d sessions, cleaned %d worktrees.\n", killed, cleaned)
}

func runDebug() {
	store := session.NewStore("")
	fmt.Println("State file:", store.Path())
	fmt.Println("Version:", version)
}

func runReset() {
	store := session.NewStore("")
	os.Remove(store.Path())

	// Clear autonomous-runtime transient state as well. Leave config and
	// history intact — those are user-owned.
	executor := exec.NewRealExecutor()
	wtMgr := worktree.NewManager(executor)
	cwd, _ := os.Getwd()
	root, err := wtMgr.ProjectRoot(nil, cwd)
	if err == nil {
		ck := filepath.Join(root, ".cavekit")
		for _, name := range []string{".loop.json", ".loop.lock", ".progress.json", ".auto-backprop-pending.json", ".debug.log"} {
			os.Remove(filepath.Join(ck, name))
		}
	}
	fmt.Println("State cleared.")
}

func preflight(program string) error {
	if _, err := osexec.LookPath("tmux"); err != nil {
		return fmt.Errorf("tmux not installed")
	}
	if _, err := osexec.LookPath("git"); err != nil {
		return fmt.Errorf("git not installed")
	}
	if _, err := osexec.LookPath(program); err != nil {
		return fmt.Errorf("%s not installed (use --program to override)", program)
	}
	return nil
}
