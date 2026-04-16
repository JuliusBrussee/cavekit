package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var allowlistExecutables = []string{
	"ls", "cat", "head", "tail", "less", "more", "wc", "file", "stat", "du", "df",
	"pwd", "whoami", "id", "uname", "hostname", "date",
	"echo", "printf", "true", "false",
	"grep", "rg", "ag", "ack", "find", "fd", "locate", "which", "where", "type",
	"git",
	"node", "npm", "npx", "yarn", "pnpm", "bun", "deno", "tsx", "ts-node",
	"python", "python3", "pip", "pip3", "uv",
	"go", "cargo", "rustc", "rustup",
	"make", "cmake",
	"docker",
	"kubectl", "helm",
	"jq", "yq", "sed", "awk", "sort", "uniq", "tr", "cut", "paste",
	"curl", "wget",
	"ssh", "scp",
	"tar", "gzip", "gunzip", "zip", "unzip",
	"diff", "patch",
	"test", "[", "[[",
	"tput", "clear", "reset",
	"codex",
}

var allowlistGit = []string{
	"status", "log", "diff", "show", "branch", "tag", "remote", "stash",
	"fetch", "pull",
	"add", "commit",
	"checkout", "switch",
	"rebase", "merge",
	"cherry-pick",
	"bisect", "blame", "annotate", "shortlog", "describe",
	"ls-files", "ls-tree", "rev-parse", "rev-list", "name-rev",
	"config",
}

var blocklistPatterns = []string{
	`rm\s+(-[a-zA-Z]*r[a-zA-Z]*f|--recursive\b.*--force|-[a-zA-Z]*f[a-zA-Z]*r)\b.*(/|\*|\.\.|~)`,
	`git\s+push\s+.*--force\b.*\b(main|master)\b`,
	`git\s+push\s+.*-f\b.*\b(main|master)\b`,
	`git\s+reset\s+--hard`,
	`git\s+clean\s+-[a-zA-Z]*f`,
	`\b(DROP|TRUNCATE|DELETE\s+FROM)\b.*\b(TABLE|DATABASE)\b`,
	`curl\b.*\|\s*(bash|sh|zsh)\b`,
	`wget\b.*\|\s*(bash|sh|zsh)\b`,
	`chmod\s+777\b`,
	`chmod\s+-R\s+777\b`,
	`:\(\)\s*\{\s*:\|:\s*&\s*\}\s*;`,
	`mkfs\b`,
	`dd\s+.*of=/dev/`,
	`\bsudo\s+rm\b`,
}

var blocklistGit = []string{
	`push.*--force`,
	`push.*-f\b`,
	`reset\s+--hard`,
	`clean\s+-[a-zA-Z]*f`,
	`branch\s+-D`,
}

var (
	quotedDoubleRE = regexp.MustCompile(`"[^"]*"`)
	quotedSingleRE = regexp.MustCompile(`'[^']*'`)
	substRE        = regexp.MustCompile(`\$\([^)]*\)`)
	varRE          = regexp.MustCompile(`\$\{[^}]*\}`)
	pathAbsRE      = regexp.MustCompile(`(/[a-zA-Z0-9_./-]+)`)
	pathRelRE      = regexp.MustCompile(`\./[a-zA-Z0-9_./-]+`)
	hashRE         = regexp.MustCompile(`\b[0-9a-f]{7,40}\b`)
)

type gateHookInput struct {
	ToolName string `json:"tool_name"`
	Command  string `json:"command"`
	Input    struct {
		Command string `json:"command"`
	} `json:"input"`
}

func runCommandGate(args []string) {
	cmd := "hook"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	switch cmd {
	case "hook":
		exitOnError(runCommandGateHook(args))
	case "classify":
		if len(args) == 0 {
			exitOnError(fmt.Errorf("cavekit command-gate classify: command required"))
		}
		cfg := newCavekitConfig(currentProjectRootOrCwd())
		result, err := fastClassify(strings.Join(args, " "), cfg)
		exitOnError(err)
		fmt.Println(result)
	case "normalize":
		if len(args) == 0 {
			exitOnError(fmt.Errorf("cavekit command-gate normalize: command required"))
		}
		fmt.Println(normalizeGateCommand(strings.Join(args, " ")))
	case "codex":
		if len(args) == 0 {
			exitOnError(fmt.Errorf("cavekit command-gate codex: command required"))
		}
		cfg := newCavekitConfig(currentProjectRootOrCwd())
		result, err := codexClassify(strings.Join(args, " "), currentProjectRootOrCwd(), cfg)
		exitOnError(err)
		fmt.Println(result)
	case "cache-clear":
		exitOnError(os.RemoveAll(gateCacheFile()))
	case "help", "--help", "-h":
		fmt.Println(commandGateUsage())
	default:
		exitOnError(fmt.Errorf("unknown cavekit command-gate mode: %s", cmd))
	}
}

func runCommandGateHook(args []string) error {
	cfg := newCavekitConfig(currentProjectRootOrCwd())
	toolName, command, err := parseGateHookInput(args)
	if err != nil {
		return err
	}
	if !strings.EqualFold(toolName, "bash") || strings.TrimSpace(command) == "" {
		return nil
	}

	gateMode, err := cfg.Get("command_gate", "all")
	if err != nil {
		return err
	}
	if gateMode == "off" {
		return nil
	}

	if os.Getenv("BP_HOOK_ALREADY_ALLOWED") == "1" || os.Getenv("BP_HOOK_ALREADY_BLOCKED") == "1" {
		return nil
	}

	fastResult, err := fastClassify(command, cfg)
	if err != nil {
		return err
	}
	switch {
	case fastResult == "APPROVE":
		return nil
	case strings.HasPrefix(fastResult, "BLOCK|"):
		return printGateDecision("block", strings.TrimPrefix(fastResult, "BLOCK|"))
	}

	normalized := normalizeGateCommand(command)
	if cached := gateCacheGet(normalized); cached != "" {
		switch {
		case strings.HasPrefix(cached, "APPROVE"), strings.HasPrefix(cached, "PASSTHROUGH"):
			return nil
		case strings.HasPrefix(cached, "BLOCK|"):
			return printGateDecision("block", strings.TrimPrefix(cached, "BLOCK|"))
		}
	}

	codexResult, err := codexClassify(command, currentProjectRootOrCwd(), cfg)
	if err != nil {
		return err
	}
	gateCacheSet(normalized, codexResult)

	if strings.HasPrefix(codexResult, "BLOCK|") {
		return printGateDecision("block", strings.TrimPrefix(codexResult, "BLOCK|"))
	}
	return nil
}

func parseGateHookInput(args []string) (string, string, error) {
	if len(args) >= 2 {
		return args[0], strings.Join(args[1:], " "), nil
	}

	stdin, err := ioReadAll(os.Stdin)
	if err != nil {
		return "", "", err
	}
	if len(strings.TrimSpace(stdin)) == 0 {
		return "", "", nil
	}

	var payload gateHookInput
	if err := json.Unmarshal([]byte(stdin), &payload); err == nil {
		command := payload.Command
		if command == "" {
			command = payload.Input.Command
		}
		return payload.ToolName, command, nil
	}

	var generic map[string]any
	if err := json.Unmarshal([]byte(stdin), &generic); err != nil {
		return "", "", nil
	}
	toolName, _ := generic["tool_name"].(string)
	command, _ := generic["command"].(string)
	if command == "" {
		if input, ok := generic["input"].(map[string]any); ok {
			command, _ = input["command"].(string)
		}
	}
	return toolName, command, nil
}

func normalizeGateCommand(cmd string) string {
	normalized := quotedDoubleRE.ReplaceAllString(cmd, "<STR>")
	normalized = quotedSingleRE.ReplaceAllString(normalized, "<STR>")
	normalized = substRE.ReplaceAllString(normalized, "<SUBST>")
	normalized = varRE.ReplaceAllString(normalized, "<VAR>")
	normalized = pathAbsRE.ReplaceAllString(normalized, "<PATH>")
	normalized = pathRelRE.ReplaceAllString(normalized, "<PATH>")
	normalized = hashRE.ReplaceAllString(normalized, "<HASH>")
	return strings.Join(strings.Fields(normalized), " ")
}

func fastClassify(cmd string, cfg *cavekitConfig) (string, error) {
	baseExec := ""
	fields := strings.Fields(cmd)
	if len(fields) > 0 {
		baseExec = filepath.Base(fields[0])
	}

	for _, pattern := range blocklistPatterns {
		if matched, _ := regexp.MatchString(pattern, cmd); matched {
			return "BLOCK|Matches blocklist pattern: " + pattern, nil
		}
	}

	userBlocklist, err := cfg.Get("command_gate_blocklist", "")
	if err != nil {
		return "", err
	}
	for _, entry := range strings.Fields(userBlocklist) {
		if matched, _ := regexp.MatchString(entry, cmd); matched {
			return "BLOCK|Matches user blocklist: " + entry, nil
		}
	}

	if baseExec == "git" && len(fields) > 1 {
		for _, pattern := range blocklistGit {
			if matched, _ := regexp.MatchString(`git\s+`+pattern, cmd); matched {
				return "BLOCK|Dangerous git operation: " + pattern, nil
			}
		}
		for _, allowed := range allowlistGit {
			if fields[1] == allowed {
				return "APPROVE", nil
			}
		}
	}

	for _, allowed := range allowlistExecutables {
		if baseExec == allowed {
			return "APPROVE", nil
		}
	}

	userAllowlist, err := cfg.Get("command_gate_allowlist", "")
	if err != nil {
		return "", err
	}
	for _, entry := range strings.Fields(userAllowlist) {
		if baseExec == entry {
			return "APPROVE", nil
		}
		if matched, _ := regexp.MatchString(entry, cmd); matched {
			return "APPROVE", nil
		}
	}

	return "UNKNOWN", nil
}

func codexClassify(cmd, workdir string, cfg *cavekitConfig) (string, error) {
	status := detectCodex(effectiveHomeDir())
	if !status.Available {
		return "PASSTHROUGH|Codex unavailable", nil
	}

	timeoutValue, err := cfg.Get("command_gate_timeout", "3000")
	if err != nil {
		return "", err
	}
	timeoutMs := 3000
	if timeoutValue != "" {
		if parsed, parseErr := strconv.Atoi(timeoutValue); parseErr == nil && parsed > 0 {
			timeoutMs = parsed
		}
	}

	model, err := cfg.Get("command_gate_model", "o4-mini")
	if err != nil {
		return "", err
	}
	if model == "" {
		model = "o4-mini"
	}

	prompt := strings.Join([]string{
		"You are a command safety classifier. Given a bash command and its working directory context, classify whether it is safe to execute autonomously. Output exactly one JSON object with these fields:",
		"- safe: boolean (true if the command is safe)",
		"- reason: string (brief explanation)",
		"- severity: \"info\" | \"warn\" | \"block\"",
		"",
		"Rules:",
		"- \"info\": safe to run silently",
		"- \"warn\": probably safe but worth logging (e.g., writes to important files)",
		"- \"block\": potentially destructive or dangerous (data loss, credential exposure, network exfiltration)",
		"",
		"Be conservative: if unsure, classify as \"warn\" not \"info\".",
		"Do NOT block standard development commands (test runners, build tools, linters, formatters).",
		"DO block commands that delete data, expose secrets, or modify system configuration.",
		"",
		"Command: " + cmd,
		"Working directory: " + workdir,
	}, "\n")

	raw, err := runCodexPrompt(prompt, model, time.Duration(timeoutMs)*time.Millisecond, "")
	if err != nil {
		return "PASSTHROUGH|Codex classification failed", nil
	}

	var response codexReviewResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &response); err != nil {
		return "PASSTHROUGH|Could not parse Codex response", nil
	}

	switch response.Severity {
	case "info":
		return "APPROVE|" + response.Reason, nil
	case "warn":
		return "APPROVE|" + response.Reason, nil
	case "block":
		return "BLOCK|" + response.Reason, nil
	default:
		if response.Safe {
			return "APPROVE|" + response.Reason, nil
		}
		return "BLOCK|" + response.Reason, nil
	}
}

func gateCacheFile() string {
	if path := os.Getenv("BP_GATE_CACHE_FILE"); path != "" {
		return path
	}
	return filepath.Join(os.TempDir(), "bp-gate-cache")
}

func gateCacheGet(key string) string {
	data, err := os.ReadFile(gateCacheFile())
	if err != nil {
		return ""
	}
	var value string
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		prefix := key + "|"
		if strings.HasPrefix(line, prefix) {
			value = strings.TrimPrefix(line, prefix)
		}
	}
	return value
}

func gateCacheSet(key, value string) {
	if err := ensureDir(filepath.Dir(gateCacheFile())); err != nil {
		return
	}
	f, err := os.OpenFile(gateCacheFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s|%s\n", key, value)
}

func printGateDecision(decision, reason string) error {
	payload := map[string]string{
		"decision": decision,
		"reason":   reason,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	fmt.Println(string(raw))
	return nil
}

func commandGateUsage() string {
	lines := []string{
		"Usage: cavekit command-gate {hook|classify|normalize|codex|cache-clear}",
		"  hook [tool cmd]   Run as PreToolUse hook (or reads JSON stdin)",
		"  classify <cmd>    Fast-path classify a command",
		"  normalize <cmd>   Normalize command for caching",
		"  codex <cmd>       Send to Codex for classification",
		"  cache-clear       Clear the verdict cache",
	}
	return strings.Join(lines, "\n")
}

func ioReadAll(f *os.File) (string, error) {
	data, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
