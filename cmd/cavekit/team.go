package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/JuliusBrussee/cavekit/internal/exec"
	"github.com/JuliusBrussee/cavekit/internal/team"
	"github.com/JuliusBrussee/cavekit/internal/worktree"
)

func runTeam() {
	if len(os.Args) < 3 {
		fmt.Fprint(os.Stderr, teamUsage(false))
		os.Exit(2)
	}

	subcmd := os.Args[2]
	if subcmd == "-h" || subcmd == "--help" {
		fmt.Print(teamUsage(true))
		return
	}

	root := mustProjectRoot()
	manager := team.NewManager(root, exec.NewRealExecutor(), os.Stderr)
	ctx := context.Background()

	switch subcmd {
	case "init":
		runTeamInit(ctx, manager, os.Args[3:])
	case "join":
		runTeamJoin(ctx, manager, os.Args[3:])
	case "status":
		runTeamStatus(manager, os.Args[3:])
	case "claim":
		runTeamClaim(ctx, manager, os.Args[3:])
	case "release":
		runTeamRelease(ctx, manager, os.Args[3:])
	case "sync":
		runTeamSync(ctx, manager, os.Args[3:])
	case "next":
		runTeamNext(ctx, manager, os.Args[3:])
	case "guard-commit":
		runTeamGuardCommit(ctx, manager)
	case "heartbeat":
		if os.Getenv("CAVEKIT_INTERNAL") == "" {
			fmt.Fprintln(os.Stderr, "unknown team subcommand: heartbeat")
			os.Exit(2)
		}
		runTeamHeartbeat(ctx, manager, os.Args[3:])
	default:
		fmt.Fprintf(os.Stderr, "unknown team subcommand: %s\n", subcmd)
		os.Exit(2)
	}
}

func runTeamInit(ctx context.Context, manager *team.Manager, args []string) {
	args = normalizeFlagArgs(args, map[string]bool{
		"--email": true,
		"--name":  true,
	})
	fs := flag.NewFlagSet("team init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	force := fs.Bool("force", false, "rewrite scaffolding without touching the roster")
	email := fs.String("email", "", "explicit git email to use for identity")
	name := fs.String("name", "", "explicit display name to use for identity")
	jsonOut := fs.Bool("json", false, "print JSON only")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	result, err := manager.Init(ctx, *force, *email, *name)
	exitIfTeamErr(err)
	for _, warning := range result.Warnings {
		fmt.Fprintln(os.Stderr, warning)
	}
	if *jsonOut {
		writeJSONResult(result)
		return
	}
	fmt.Printf("initialized team mode at %s\n", team.TeamDir(manager.Root))
	fmt.Printf("identity: %s\n", result.Identity.Email)
	if result.RosterCreated {
		fmt.Printf("created roster: %s\n", team.RosterPath(manager.Root))
	}
	if result.RefReady {
		fmt.Printf("ledger ref: %s (ready)\n", team.TeamRef)
	} else {
		fmt.Printf("ledger ref: %s (deferred — push on next online op)\n", team.TeamRef)
	}
	fmt.Println("pre-commit guard installed at .git/hooks/pre-commit")
}

func runTeamJoin(ctx context.Context, manager *team.Manager, args []string) {
	args = normalizeFlagArgs(args, map[string]bool{
		"--email": true,
		"--name":  true,
	})
	fs := flag.NewFlagSet("team join", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	email := fs.String("email", "", "explicit git email to use for identity")
	name := fs.String("name", "", "explicit display name to use for identity")
	strict := fs.Bool("strict", false, "exit 3 instead of succeeding when already joined")
	jsonOut := fs.Bool("json", false, "print JSON only")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	result, err := manager.Join(ctx, *email, *name, *strict)
	exitIfTeamErr(err)
	if *jsonOut {
		writeJSONResult(result)
		return
	}
	if result.Already {
		fmt.Printf("already joined: %s\n", result.Identity.Email)
		return
	}
	fmt.Printf("joined team as %s\n", result.Identity.Email)
}

func runTeamStatus(manager *team.Manager, args []string) {
	args = normalizeFlagArgs(args, map[string]bool{
		"--task": true,
		"--user": true,
	})
	fs := flag.NewFlagSet("team status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	taskID := fs.String("task", "", "restrict output to one task")
	user := fs.String("user", "", "restrict output to one teammate email")
	conflicts := fs.Bool("conflicts", false, "include recent conflict/race events and provisional claims")
	jsonOut := fs.Bool("json", false, "print JSON only")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	var identity *team.Identity
	if id, err := team.ReadIdentity(manager.Root); err == nil {
		identity = &id
	}
	report, err := team.BuildStatusReport(manager.Root, identity, strings.TrimSpace(*taskID), strings.TrimSpace(*user), os.Stderr)
	exitIfTeamErr(err)
	if *conflicts {
		report.Conflicts = team.CollectConflicts(manager.Root, os.Stderr)
		report.OutboxPending = team.OutboxPendingCount(manager.Root)
	}
	if *jsonOut {
		writeJSONResult(report)
		return
	}
	fmt.Print(team.FormatStatusReport(report))
	if *conflicts {
		fmt.Print(team.FormatConflicts(report))
	}
}

func runTeamClaim(ctx context.Context, manager *team.Manager, args []string) {
	args = normalizeFlagArgs(args, map[string]bool{
		"--paths": true,
	})
	fs := flag.NewFlagSet("team claim", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	paths := fs.String("paths", "", "comma-separated file globs this claim owns (e.g. 'src/auth/**,tests/auth/**')")
	jsonOut := fs.Bool("json", false, "print JSON only")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: cavekit team claim <task-id> [--paths GLOBS] [--json]")
		os.Exit(2)
	}

	pathList := splitCSV(*paths)
	result, err := manager.Claim(ctx, strings.TrimSpace(fs.Arg(0)), pathList)
	exitIfTeamErr(err)
	if *jsonOut {
		writeJSONResult(result)
		return
	}
	if result.Already {
		fmt.Printf("already claimed: %s\n", result.Task)
		return
	}
	fmt.Printf("claimed %s", result.Task)
	if len(result.Paths) > 0 {
		fmt.Printf(" (paths: %s)", strings.Join(result.Paths, ", "))
	}
	fmt.Println()
	if result.Provisional {
		fmt.Println("provisional: queued in outbox (offline); will publish on next successful op")
	}
	if result.CommitSHA != "" {
		fmt.Printf("ledger: %s\n", result.CommitSHA)
	}
}

func runTeamRelease(ctx context.Context, manager *team.Manager, args []string) {
	args = normalizeFlagArgs(args, map[string]bool{
		"--note": true,
	})
	fs := flag.NewFlagSet("team release", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	note := fs.String("note", "", "note to record alongside the release")
	complete := fs.Bool("complete", false, "record a complete event instead of a release event")
	jsonOut := fs.Bool("json", false, "print JSON only")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: cavekit team release <task-id> [--note TEXT] [--complete] [--json]")
		os.Exit(2)
	}

	result, err := manager.Release(ctx, strings.TrimSpace(fs.Arg(0)), *note, *complete)
	exitIfTeamErr(err)
	if *jsonOut {
		writeJSONResult(result)
		return
	}
	if result.Noop {
		fmt.Printf("not currently claimed: %s\n", result.Task)
		return
	}
	action := "released"
	if result.Complete {
		action = "completed"
	}
	fmt.Printf("%s %s\n", action, result.Task)
	if result.Provisional {
		fmt.Println("provisional: queued in outbox")
	}
	if result.CommitSHA != "" {
		fmt.Printf("ledger: %s\n", result.CommitSHA)
	}
}

func runTeamSync(ctx context.Context, manager *team.Manager, args []string) {
	args = normalizeFlagArgs(args, map[string]bool{
		"--timeout": true,
	})
	fs := flag.NewFlagSet("team sync", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	timeout := fs.Int("timeout", 10, "fetch timeout in seconds")
	jsonOut := fs.Bool("json", false, "print JSON only")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	result, err := manager.Sync(ctx, *timeout)
	exitIfTeamErr(err)
	if *jsonOut {
		writeJSONResult(result)
		return
	}
	fmt.Printf("fetched team state (%d events", result.EventCount)
	if result.OutboxPending > 0 {
		fmt.Printf(", %d queued offline", result.OutboxPending)
	}
	fmt.Println(")")
}

func runTeamNext(ctx context.Context, manager *team.Manager, args []string) {
	_ = ctx
	args = normalizeFlagArgs(args, nil)
	fs := flag.NewFlagSet("team next", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "print JSON only")
	_ = fs.Parse(args)

	identity, err := team.ReadIdentity(manager.Root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cavekit team next requires `team join` first")
		os.Exit(1)
	}
	suggestion, err := team.NextTask(manager.Root, identity, os.Stderr)
	exitIfTeamErr(err)
	if *jsonOut {
		writeJSONResult(suggestion)
		return
	}
	if suggestion.Task == nil {
		fmt.Println("no unclaimed frontier task available for you right now")
		if len(suggestion.SkippedBy) > 0 {
			fmt.Println("blocked candidates:")
			for id, reason := range suggestion.SkippedBy {
				fmt.Printf("  %s — %s\n", id, reason)
			}
		}
		return
	}
	fmt.Printf("next: %s — %s (tier %d)\n", suggestion.Task.ID, suggestion.Task.Title, suggestion.Task.Tier)
	if len(suggestion.Alternatives) > 0 {
		fmt.Println("alternatives:")
		for i, alt := range suggestion.Alternatives {
			if i >= 3 {
				break
			}
			fmt.Printf("  %s — %s\n", alt.ID, alt.Title)
		}
	}
}

func runTeamGuardCommit(ctx context.Context, manager *team.Manager) {
	if err := team.GuardCommit(ctx, manager.Root, exec.NewRealExecutor(), os.Stderr); err != nil {
		if exitErr, ok := err.(*team.ExitError); ok {
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runTeamHeartbeat(ctx context.Context, manager *team.Manager, args []string) {
	args = normalizeFlagArgs(args, map[string]bool{
		"--interval": true,
	})
	fs := flag.NewFlagSet("team heartbeat", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	interval := fs.Int("interval", 0, "override heartbeat interval in seconds")
	once := fs.Bool("once", false, "write a single heartbeat tick then exit")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: cavekit team heartbeat <task-id> [--interval SECONDS] [--once]")
		os.Exit(2)
	}

	err := manager.Heartbeat(ctx, strings.TrimSpace(fs.Arg(0)), time.Duration(*interval)*time.Second, *once)
	exitIfTeamErr(err)
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func teamUsage(showHidden bool) string {
	var b strings.Builder
	b.WriteString("usage: cavekit team <subcommand> [flags]\n\n")
	b.WriteString("subcommands:\n")
	b.WriteString("  init          initialize team scaffolding, identity, ledger ref, and pre-commit guard\n")
	b.WriteString("  join          resolve and persist local identity for an initialized team\n")
	b.WriteString("  status        show active claims, recent activity, and idle members\n")
	b.WriteString("  claim         claim one frontier task for the local identity (optional --paths)\n")
	b.WriteString("  release       release or complete a claimed task\n")
	b.WriteString("  sync          fetch the team ledger ref and refresh the local cache\n")
	b.WriteString("  next          suggest the best unclaimed frontier task for you\n")
	b.WriteString("  guard-commit  (invoked by pre-commit hook) block commits that touch others' claims\n")
	if showHidden && os.Getenv("CAVEKIT_INTERNAL") != "" {
		b.WriteString("  heartbeat     internal lease refresh loop used by /ck:make\n")
	}
	return b.String()
}

func mustProjectRoot() string {
	executor := exec.NewRealExecutor()
	wtMgr := worktree.NewManager(executor)
	root, err := wtMgr.ProjectRoot(context.Background(), mustGetwd())
	if err != nil {
		fmt.Fprintf(os.Stderr, "not in a git repo: %s\n", err)
		os.Exit(1)
	}
	return root
}

func mustGetwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
		os.Exit(1)
	}
	return cwd
}

func writeJSONResult(value any) {
	data, err := json.Marshal(value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal json: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s\n", data)
}

func exitIfTeamErr(err error) {
	if err == nil {
		return
	}
	if exitErr, ok := err.(*team.ExitError); ok {
		if exitErr.Message != "" {
			fmt.Fprintln(os.Stderr, exitErr.Message)
		}
		os.Exit(exitErr.Code)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func normalizeFlagArgs(args []string, flagsWithValue map[string]bool) []string {
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if flagsWithValue[arg] && !strings.Contains(arg, "=") && i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		positionals = append(positionals, arg)
	}
	return append(flags, positionals...)
}
