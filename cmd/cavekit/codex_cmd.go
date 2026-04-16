package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	osexec "os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type codexStatus struct {
	BinaryAvailable bool
	PluginPresent   bool
	Available       bool
}

type localMarketplace struct {
	Name      string                  `json:"name"`
	Interface map[string]string       `json:"interface"`
	Plugins   []localMarketplaceEntry `json:"plugins"`
}

type localMarketplaceEntry struct {
	Name     string                    `json:"name"`
	Source   localMarketplaceSource    `json:"source"`
	Policy   localMarketplacePolicy    `json:"policy"`
	Category string                    `json:"category"`
}

type localMarketplaceSource struct {
	Source string `json:"source"`
	Path   string `json:"path"`
}

type localMarketplacePolicy struct {
	Installation string `json:"installation"`
	Authentication string `json:"authentication"`
}

type codexReviewResponse struct {
	Safe     bool   `json:"safe"`
	Reason   string `json:"reason"`
	Severity string `json:"severity"`
}

func detectCodex(homeDir string) codexStatus {
	status := codexStatus{}

	if codexPath, err := osexec.LookPath("codex"); err == nil {
		cmd := osexec.Command(codexPath, "--version")
		if err := cmd.Run(); err == nil {
			status.BinaryAvailable = true
		}
	}

	candidateRoots := []string{
		filepath.Join(homeDir, ".codex"),
		filepath.Join(homeDir, ".claude", "plugins"),
		filepath.Join(homeDir, ".claude", "plugins", "local"),
	}
	for _, root := range candidateRoots {
		if codexPluginPresent(root) {
			status.PluginPresent = true
			break
		}
	}

	status.Available = status.BinaryAvailable
	return status
}

func codexPluginPresent(root string) bool {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return false
	}
	if strings.EqualFold(filepath.Base(root), ".codex") {
		return true
	}

	found := false
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr == nil && strings.Count(rel, string(os.PathSeparator)) > 2 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(d.Name())
		if strings.Contains(name, "codex") {
			found = true
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

func runSyncCodex(args []string) {
	sourceDir, remaining, err := parseSourceDirFlag(args)
	exitOnError(err)
	if len(remaining) > 0 {
		exitOnError(fmt.Errorf("cavekit sync-codex: unknown arguments: %s", strings.Join(remaining, " ")))
	}

	sourceRoot := resolveSourceDir(sourceDir)
	exitOnError(syncCodexPlugin(sourceRoot, effectiveHomeDir()))
	fmt.Println("Codex sync complete.")
}

func syncCodexPlugin(sourceRoot, homeDir string) error {
	pluginsDir := filepath.Join(homeDir, "plugins")
	pluginDir := filepath.Join(pluginsDir, "ck")
	marketplaceFile := filepath.Join(homeDir, ".agents", "plugins", "marketplace.json")
	legacyLink := filepath.Join(homeDir, ".codex", "cavekit")
	promptsDir := filepath.Join(homeDir, ".codex", "prompts")

	if err := ensureDir(pluginsDir); err != nil {
		return err
	}
	if err := ensureDir(filepath.Dir(marketplaceFile)); err != nil {
		return err
	}
	if err := ensureDir(promptsDir); err != nil {
		return err
	}

	if err := createDirLink(sourceRoot, pluginDir); err != nil {
		return fmt.Errorf("link plugin: %w", err)
	}

	if dirExists(filepath.Join(homeDir, ".codex")) {
		if err := createDirLink(sourceRoot, legacyLink); err != nil {
			return fmt.Errorf("link legacy codex shortcut: %w", err)
		}
	}

	commandsDir := filepath.Join(sourceRoot, "commands")
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		return fmt.Errorf("read commands: %w", err)
	}

	commandNames := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		commandName := strings.TrimSuffix(entry.Name(), ".md")
		commandNames[commandName] = struct{}{}
		src := filepath.Join(commandsDir, entry.Name())
		for _, prefix := range []string{"ck", "bp"} {
			dst := filepath.Join(promptsDir, prefix+"-"+commandName+".md")
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("copy prompt %s: %w", commandName, err)
			}
		}
	}

	for _, prefix := range []string{"ck", "bp"} {
		matches, err := filepath.Glob(filepath.Join(promptsDir, prefix+"-*.md"))
		if err != nil {
			return err
		}
		for _, match := range matches {
			name := strings.TrimSuffix(filepath.Base(match), ".md")
			commandName := strings.TrimPrefix(name, prefix+"-")
			if _, ok := commandNames[commandName]; !ok {
				if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
					return err
				}
			}
		}
	}

	if err := updateLocalMarketplace(marketplaceFile); err != nil {
		return err
	}

	return nil
}

func updateLocalMarketplace(path string) error {
	entry := localMarketplaceEntry{
		Name: "ck",
		Source: localMarketplaceSource{
			Source: "local",
			Path:   "./plugins/ck",
		},
		Policy: localMarketplacePolicy{
			Installation: "AVAILABLE",
			Authentication: "ON_INSTALL",
		},
		Category: "Productivity",
	}

	data := localMarketplace{
		Name:      "local-plugins",
		Interface: map[string]string{"displayName": "Local Plugins"},
	}

	if fileExists(path) {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return err
		}
		if data.Interface == nil {
			data.Interface = map[string]string{"displayName": "Local Plugins"}
		}
	}

	replaced := false
	for i, plugin := range data.Plugins {
		if plugin.Name == "ck" || plugin.Name == "bp" {
			data.Plugins[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		data.Plugins = append(data.Plugins, entry)
	}

	return writeJSONFile(path, data)
}

func runCodexReview(args []string) {
	projectRoot := currentProjectRootOrCwd()
	cfg := newCavekitConfig(projectRoot)

	baseRef := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--base":
			if i+1 >= len(args) {
				exitOnError(errors.New("cavekit codex-review: --base requires a value"))
			}
			baseRef = args[i+1]
			i++
		case "--help", "-h":
			fmt.Println(codexReviewUsage())
			return
		default:
			exitOnError(fmt.Errorf("cavekit codex-review: unknown argument %s", args[i]))
		}
	}

	reviewMode, err := cfg.Get("codex_review", "auto")
	exitOnError(err)
	if reviewMode == "off" {
		fmt.Println("[ck:review] Codex review is disabled (codex_review=off). Skipping.")
		return
	}

	status := detectCodex(effectiveHomeDir())
	if !status.Available {
		fmt.Println("[ck:review] Codex is not available. Falling back to inspector-only review.")
		return
	}

	if baseRef == "" {
		baseRef, err = detectBaseRef(projectRoot)
		exitOnError(err)
	}

	fmt.Printf("[ck:review] Computing diff %s...HEAD\n", baseRef)
	diff, err := gitDiff(projectRoot, baseRef)
	exitOnError(err)
	if strings.TrimSpace(diff) == "" {
		fmt.Println("[ck:review] No diff found. Nothing to review.")
		return
	}

	diffLines := 0
	if trimmed := strings.TrimSpace(diff); trimmed != "" {
		diffLines = len(strings.Split(trimmed, "\n"))
	}
	fmt.Printf("[ck:review] Diff is %d lines. Sending to Codex...\n", diffLines)

	model, err := cfg.Get("codex_model", "o4-mini")
	exitOnError(err)
	if model == "" {
		model = "o4-mini"
	}

	if os.Getenv("BP_CODEX_DRY_RUN") == "1" {
		fmt.Printf("[ck:review] DRY RUN - would execute: codex --approval-mode full-auto --model %s --quiet -p <prompt> <<< <diff>\n", model)
		return
	}

	prompt, err := buildReviewPrompt(cfg)
	exitOnError(err)
	rawOutput, err := runCodexPrompt(prompt, model, 0, diff)
	if err != nil {
		fmt.Println("[ck:review] Codex invocation failed. Falling back to inspector-only review.")
		fmt.Printf("[ck:review] Error: %.500s\n", err.Error())
		return
	}

	if strings.Contains(strings.ToUpper(rawOutput), "NO_FINDINGS") {
		fmt.Println("[ck:review] Codex found no issues. Clean review.")
		return
	}

	findingsFile := filepath.Join(projectRoot, "context", "impl", "impl-review-findings.md")
	findings, err := parseCodexFindings(rawOutput, findingsFile)
	if err != nil {
		exitOnError(err)
	}
	if strings.TrimSpace(findings) == "" {
		fmt.Println("[ck:review] Could not parse findings from Codex output.")
		fmt.Printf("[ck:review] Raw (first 1000 chars): %.1000s\n", rawOutput)
		return
	}

	exitOnError(appendFindings(findingsFile, findings))

	fmt.Println()
	fmt.Println("[ck:review] === Codex Adversarial Review Findings ===")
	fmt.Print(findings)
	if !strings.HasSuffix(findings, "\n") {
		fmt.Println()
	}
	fmt.Println("[ck:review] === End of Findings ===")
	fmt.Printf("[ck:review] Findings appended to %s\n", findingsFile)
}

func buildReviewPrompt(cfg *cavekitConfig) (string, error) {
	caveman, err := cfg.CavemanActive("build")
	if err != nil {
		return "", err
	}
	if caveman {
		return "Senior engineer. Adversarial code review. Check diff for bugs, security holes, logic errors, spec violations. Each finding = one row in markdown table: Severity, File, Line, Description. Severity: P0 (critical) | P1 (high) | P2 (medium) | P3 (low). No issues found = output NO_FINDINGS alone.", nil
	}
	return "You are a senior engineer performing adversarial code review. Review the following diff for bugs, security issues, logic errors, and spec violations. For each finding output exactly one row in a markdown table with columns: Severity, File, Line, Description. Severity must be one of P0 (critical), P1 (high), P2 (medium), P3 (low). If no issues found, output exactly the word NO_FINDINGS on its own line and nothing else.", nil
}

func detectBaseRef(projectRoot string) (string, error) {
	upstream, err := gitOutput(projectRoot, "rev-parse", "--abbrev-ref", "@{upstream}")
	if err == nil && strings.TrimSpace(upstream) != "" {
		return strings.TrimSpace(upstream), nil
	}

	for _, candidate := range []string{"main", "master", "develop"} {
		if _, err := gitOutput(projectRoot, "rev-parse", "--verify", candidate); err == nil {
			return candidate, nil
		}
	}
	return "HEAD~10", nil
}

func gitDiff(projectRoot, baseRef string) (string, error) {
	diff, err := gitOutput(projectRoot, "diff", baseRef+"...HEAD")
	if err == nil && strings.TrimSpace(diff) != "" {
		return diff, nil
	}
	if fallback, fallbackErr := gitOutput(projectRoot, "diff", baseRef, "HEAD"); fallbackErr == nil {
		return fallback, nil
	}
	if err != nil {
		return "", err
	}
	return diff, nil
}

func gitOutput(projectRoot string, args ...string) (string, error) {
	cmd := osexec.Command("git", args...)
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func runCodexPrompt(prompt, model string, timeout time.Duration, stdin string) (string, error) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := osexec.CommandContext(ctx, "codex", "--approval-mode", "full-auto", "--model", model, "--quiet", "-p", prompt)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return stdout.String(), nil
}

func parseCodexFindings(rawOutput, findingsFile string) (string, error) {
	nextNumber, err := nextFindingNumber(findingsFile)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	for _, line := range strings.Split(strings.ReplaceAll(rawOutput, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.Contains(trimmed, "Severity") || strings.Contains(trimmed, "---") {
			continue
		}
		if !strings.Contains(trimmed, "|") || !strings.Contains(trimmed, "P") {
			continue
		}
		parts := strings.Split(trimmed, "|")
		if len(parts) < 5 {
			continue
		}
		severity := strings.TrimSpace(strings.Trim(parts[1], "`"))
		file := strings.TrimSpace(strings.Trim(parts[2], "`"))
		lineno := strings.TrimSpace(strings.Trim(parts[3], "`"))
		description := strings.TrimSpace(strings.Trim(parts[4], "`"))
		if severity == "" || description == "" || !strings.HasPrefix(severity, "P") {
			continue
		}
		fileRef := file
		if lineno != "" && lineno != "-" && !strings.EqualFold(lineno, "N/A") {
			fileRef = fmt.Sprintf("%s:L%s", file, lineno)
		}
		builder.WriteString(fmt.Sprintf("| F-%03d: %s (source: codex) | %s | %s | NEW | - |\n", nextNumber, description, severity, fileRef))
		nextNumber++
	}
	return builder.String(), nil
}

func nextFindingNumber(findingsFile string) (int, error) {
	if !fileExists(findingsFile) {
		return 1, nil
	}

	raw, err := os.ReadFile(findingsFile)
	if err != nil {
		return 0, err
	}
	maxValue := 0
	for _, token := range strings.FieldsFunc(string(raw), func(r rune) bool {
		return r == '|' || r == ' ' || r == '\n' || r == '\r'
	}) {
		if !strings.HasPrefix(token, "F-") {
			continue
		}
		value := strings.TrimSuffix(strings.TrimPrefix(token, "F-"), ":")
		n, err := strconv.Atoi(value)
		if err == nil && n > maxValue {
			maxValue = n
		}
	}
	if maxValue == 0 {
		return 1, nil
	}
	return maxValue + 1, nil
}

func appendFindings(findingsFile, findings string) error {
	if err := ensureDir(filepath.Dir(findingsFile)); err != nil {
		return err
	}
	if !fileExists(findingsFile) {
		header := strings.Join([]string{
			"# Review Findings",
			"",
			"| Finding | Severity | File | Status | Task |",
			"|---------|----------|------|--------|------|",
			"",
		}, "\n")
		if err := os.WriteFile(findingsFile, []byte(header), 0o644); err != nil {
			return err
		}
	}

	current, err := os.ReadFile(findingsFile)
	if err != nil {
		return err
	}
	content := string(current)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += findings
	return os.WriteFile(findingsFile, []byte(content), 0o644)
}

func codexReviewUsage() string {
	lines := []string{
		"Usage: cavekit codex-review [--base <ref>]",
		"",
		"Perform adversarial code review using the Codex CLI.",
		"",
		"Options:",
		"  --base <ref>    Git ref to diff against (default: auto-detect)",
		"  --help, -h      Show this help",
		"",
		"Environment:",
		"  BP_CODEX_DRY_RUN=1    Print the command without executing",
	}
	return strings.Join(lines, "\n")
}

func sortedCommandFiles(commandsDir string) ([]string, error) {
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	return files, nil
}
