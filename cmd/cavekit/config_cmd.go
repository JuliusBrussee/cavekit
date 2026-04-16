package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type configScope string

const (
	scopeProject configScope = "project"
	scopeGlobal  configScope = "global"
)

var configKeys = []string{
	"bp_model_preset",
	"codex_review",
	"codex_model",
	"codex_effort",
	"tier_gate_mode",
	"command_gate",
	"command_gate_model",
	"command_gate_timeout",
	"command_gate_allowlist",
	"command_gate_blocklist",
	"speculative_review",
	"speculative_review_timeout",
	"caveman_mode",
	"caveman_phases",
}

type cavekitConfig struct {
	projectRoot string
	homeDir     string
}

func newCavekitConfig(projectRoot string) *cavekitConfig {
	return &cavekitConfig{
		projectRoot: projectRoot,
		homeDir:     effectiveHomeDir(),
	}
}

func (c *cavekitConfig) GlobalConfigPath() string {
	if path := os.Getenv("BP_GLOBAL_CONFIG_PATH"); path != "" {
		return path
	}
	return filepath.Join(c.homeDir, ".cavekit", "config")
}

func (c *cavekitConfig) ProjectConfigPath() string {
	if path := os.Getenv("BP_PROJECT_CONFIG_PATH"); path != "" {
		return path
	}
	root := c.projectRoot
	if root == "" {
		root = currentProjectRootOrCwd()
	}
	return filepath.Join(root, ".cavekit", "config")
}

func configDefault(key string) string {
	switch key {
	case "bp_model_preset":
		return "quality"
	case "codex_review":
		return "auto"
	case "tier_gate_mode":
		return "severity"
	case "command_gate":
		return "all"
	case "command_gate_model":
		return "o4-mini"
	case "command_gate_timeout":
		return "3000"
	case "caveman_mode":
		return "on"
	case "caveman_phases":
		return "build,inspect"
	default:
		return ""
	}
}

func validateConfigValue(key, value string) error {
	switch key {
	case "bp_model_preset":
		return validateEnum(key, value, "expensive", "quality", "balanced", "fast")
	case "codex_review":
		return validateEnum(key, value, "auto", "off")
	case "tier_gate_mode":
		return validateEnum(key, value, "severity", "strict", "permissive", "off")
	case "command_gate":
		return validateEnum(key, value, "all", "interactive", "off")
	case "command_gate_timeout", "speculative_review_timeout":
		if value != "" && strings.IndexFunc(value, func(r rune) bool { return r < '0' || r > '9' }) != -1 {
			return fmt.Errorf("bp_config_set: invalid value %q for %q (allowed: positive integer)", value, key)
		}
	case "speculative_review":
		return validateEnum(key, value, "on", "off")
	case "caveman_mode":
		return validateEnum(key, value, "on", "off")
	case "caveman_phases":
		if value == "" {
			return nil
		}
		for _, phase := range strings.Split(value, ",") {
			switch phase {
			case "build", "inspect", "draft", "architect":
			default:
				return fmt.Errorf("bp_config_set: invalid phase %q in %q (allowed: build,inspect,draft,architect)", phase, key)
			}
		}
	}
	return nil
}

func validateEnum(key, value string, allowed ...string) error {
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return fmt.Errorf("bp_config_set: invalid value %q for %q (allowed: %s)", value, key, strings.Join(allowed, " "))
}

func readConfigLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return nil, nil
	}
	return strings.Split(content, "\n"), nil
}

func lastConfigValue(path, key string) (string, bool, error) {
	lines, err := readConfigLines(path)
	if err != nil {
		return "", false, err
	}
	var value string
	found := false
	prefix := key + "="
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			value = strings.TrimPrefix(line, prefix)
			found = true
		}
	}
	return value, found, nil
}

func (c *cavekitConfig) GetSource(key string) (string, error) {
	projectValue, found, err := lastConfigValue(c.ProjectConfigPath(), key)
	if err != nil {
		return "", err
	}
	if found && projectValue != "" {
		return string(scopeProject), nil
	}

	globalValue, found, err := lastConfigValue(c.GlobalConfigPath(), key)
	if err != nil {
		return "", err
	}
	if found && globalValue != "" {
		return string(scopeGlobal), nil
	}

	return "default", nil
}

func (c *cavekitConfig) GetSourcePath(key string) (string, error) {
	source, err := c.GetSource(key)
	if err != nil {
		return "", err
	}
	switch source {
	case string(scopeProject):
		return c.ProjectConfigPath(), nil
	case string(scopeGlobal):
		return c.GlobalConfigPath(), nil
	default:
		return "(built-in default)", nil
	}
}

func (c *cavekitConfig) Get(key, fallback string) (string, error) {
	if fallback == "" {
		fallback = configDefault(key)
	}

	projectValue, found, err := lastConfigValue(c.ProjectConfigPath(), key)
	if err != nil {
		return "", err
	}
	if found && projectValue != "" {
		return projectValue, nil
	}

	globalValue, found, err := lastConfigValue(c.GlobalConfigPath(), key)
	if err != nil {
		return "", err
	}
	if found && globalValue != "" {
		return globalValue, nil
	}

	return fallback, nil
}

func updateConfigValue(path, key, value string) error {
	lines, err := readConfigLines(path)
	if err != nil {
		return err
	}

	replaced := false
	prefix := key + "="
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			lines[i] = prefix + value
			replaced = true
		}
	}
	if !replaced {
		lines = append(lines, prefix+value)
	}

	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func (c *cavekitConfig) Set(scope configScope, key, value string) error {
	if err := validateConfigValue(key, value); err != nil {
		return err
	}

	path := c.ProjectConfigPath()
	if scope == scopeGlobal {
		path = c.GlobalConfigPath()
	}
	return updateConfigValue(path, key, value)
}

func (c *cavekitConfig) Init() error {
	if err := c.initGlobal(); err != nil {
		return err
	}
	return c.initProject()
}

func (c *cavekitConfig) initGlobal() error {
	path := c.GlobalConfigPath()
	if !fileExists(path) {
		lines := []string{
			"# Cavekit configuration",
			"# User-level defaults",
			"# See: cavekit config help",
			"",
		}
		for _, key := range configKeys {
			lines = append(lines, key+"="+configDefault(key))
		}
		if err := ensureDir(filepath.Dir(path)); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
	}

	for _, key := range configKeys {
		if _, found, err := lastConfigValue(path, key); err != nil {
			return err
		} else if !found {
			if err := updateConfigValue(path, key, configDefault(key)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *cavekitConfig) initProject() error {
	path := c.ProjectConfigPath()
	if fileExists(path) {
		return nil
	}

	lines := []string{
		"# Cavekit configuration",
		"# Project-level overrides",
		"# Add only the keys you want this repo to override.",
		"# See: cavekit config help",
	}
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func (c *cavekitConfig) EffectivePreset() (string, error) {
	preset, err := c.Get("bp_model_preset", "quality")
	if err != nil {
		return "", err
	}
	if err := validateEnum("bp_model_preset", preset, "expensive", "quality", "balanced", "fast"); err != nil {
		return "", fmt.Errorf("bp_config_effective_preset: %w", err)
	}
	return preset, nil
}

func (c *cavekitConfig) Model(taskType string) (string, error) {
	preset, err := c.EffectivePreset()
	if err != nil {
		return "", err
	}

	switch preset + ":" + taskType {
	case "expensive:reasoning", "expensive:execution", "expensive:exploration":
		return "opus", nil
	case "quality:reasoning", "quality:execution":
		return "opus", nil
	case "quality:exploration":
		return "sonnet", nil
	case "balanced:reasoning":
		return "opus", nil
	case "balanced:execution":
		return "sonnet", nil
	case "balanced:exploration", "fast:exploration":
		return "haiku", nil
	case "fast:reasoning", "fast:execution":
		return "sonnet", nil
	default:
		return "", fmt.Errorf("bp_config_model: unknown task type %q (allowed: reasoning execution exploration)", taskType)
	}
}

func (c *cavekitConfig) CavemanActive(phase string) (bool, error) {
	mode, err := c.Get("caveman_mode", "on")
	if err != nil {
		return false, err
	}
	if mode != "on" {
		return false, nil
	}

	phases, err := c.Get("caveman_phases", "build,inspect")
	if err != nil {
		return false, err
	}
	for _, candidate := range strings.Split(phases, ",") {
		if candidate == phase {
			return true, nil
		}
	}
	return false, nil
}

func (c *cavekitConfig) SummaryLine() (string, error) {
	preset, err := c.EffectivePreset()
	if err != nil {
		return "", err
	}
	reasoning, err := c.Model("reasoning")
	if err != nil {
		return "", err
	}
	execution, err := c.Model("execution")
	if err != nil {
		return "", err
	}
	exploration, err := c.Model("exploration")
	if err != nil {
		return "", err
	}
	caveman, err := c.Get("caveman_mode", "on")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"Cavekit preset: %s (reasoning=%s, execution=%s, exploration=%s, caveman=%s)",
		preset,
		reasoning,
		execution,
		exploration,
		caveman,
	), nil
}

func (c *cavekitConfig) Show() (string, error) {
	preset, err := c.EffectivePreset()
	if err != nil {
		return "", err
	}
	presetSource, err := c.GetSource("bp_model_preset")
	if err != nil {
		return "", err
	}
	presetSourcePath, err := c.GetSourcePath("bp_model_preset")
	if err != nil {
		return "", err
	}
	reasoning, err := c.Model("reasoning")
	if err != nil {
		return "", err
	}
	execution, err := c.Model("execution")
	if err != nil {
		return "", err
	}
	exploration, err := c.Model("exploration")
	if err != nil {
		return "", err
	}
	cavemanMode, err := c.Get("caveman_mode", "on")
	if err != nil {
		return "", err
	}
	cavemanPhases, err := c.Get("caveman_phases", "build,inspect")
	if err != nil {
		return "", err
	}

	lines := []string{
		"bp_model_preset=" + preset,
		"bp_model_preset_source=" + presetSource,
		"bp_model_preset_source_path=" + presetSourcePath,
		"reasoning_model=" + reasoning,
		"execution_model=" + execution,
		"exploration_model=" + exploration,
		"caveman_mode=" + cavemanMode,
		"caveman_phases=" + cavemanPhases,
		"project_config=" + c.ProjectConfigPath(),
		"global_config=" + c.GlobalConfigPath(),
	}
	return strings.Join(lines, "\n"), nil
}

func (c *cavekitConfig) List(scope configScope) (string, error) {
	path := c.ProjectConfigPath()
	if scope == scopeGlobal {
		path = c.GlobalConfigPath()
	}
	lines, err := readConfigLines(path)
	if err != nil {
		return "", err
	}

	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, "=") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			filtered = append(filtered, line)
		}
	}
	return strings.Join(filtered, "\n"), nil
}

func configPresetsTable() string {
	return strings.Join([]string{
		"| Preset | Reasoning | Execution | Exploration |",
		"|---|---|---|---|",
		"| expensive | opus | opus | opus |",
		"| quality | opus | opus | sonnet |",
		"| balanced | opus | sonnet | haiku |",
		"| fast | sonnet | sonnet | haiku |",
	}, "\n")
}

func runConfig(args []string) {
	cfg := newCavekitConfig(currentProjectRootOrCwd())

	if len(args) == 0 {
		out, err := cfg.Show()
		exitOnError(err)
		fmt.Println(out)
		return
	}

	switch args[0] {
	case "init":
		exitOnError(cfg.Init())
	case "get":
		if len(args) < 2 {
			exitOnError(errors.New("cavekit config get: key required"))
		}
		fallback := ""
		if len(args) > 2 {
			fallback = args[2]
		}
		value, err := cfg.Get(args[1], fallback)
		exitOnError(err)
		fmt.Println(value)
	case "set":
		scope := scopeProject
		trimmed := make([]string, 0, len(args)-1)
		for _, arg := range args[1:] {
			switch arg {
			case "--global":
				scope = scopeGlobal
			case "--project":
				scope = scopeProject
			default:
				trimmed = append(trimmed, arg)
			}
		}
		if len(trimmed) < 2 {
			exitOnError(errors.New("cavekit config set: key and value required"))
		}
		exitOnError(cfg.Set(scope, trimmed[0], trimmed[1]))
	case "preset":
		if len(args) < 2 {
			exitOnError(errors.New("cavekit config preset: preset name required"))
		}
		scope := scopeProject
		for _, arg := range args[2:] {
			if arg == "--global" {
				scope = scopeGlobal
			}
		}
		exitOnError(cfg.Init())
		exitOnError(cfg.Set(scope, "bp_model_preset", args[1]))
		out, err := cfg.Show()
		exitOnError(err)
		fmt.Println(out)
	case "list":
		scope := scopeProject
		for _, arg := range args[1:] {
			if arg == "--global" {
				scope = scopeGlobal
			}
		}
		out, err := cfg.List(scope)
		exitOnError(err)
		if out != "" {
			fmt.Println(out)
		}
	case "path":
		if len(args) > 1 && args[1] == "--global" {
			fmt.Println(cfg.GlobalConfigPath())
			return
		}
		fmt.Println(cfg.ProjectConfigPath())
	case "source":
		if len(args) < 2 {
			exitOnError(errors.New("cavekit config source: key required"))
		}
		out, err := cfg.GetSource(args[1])
		exitOnError(err)
		fmt.Println(out)
	case "source-path":
		if len(args) < 2 {
			exitOnError(errors.New("cavekit config source-path: key required"))
		}
		out, err := cfg.GetSourcePath(args[1])
		exitOnError(err)
		fmt.Println(out)
	case "effective-preset":
		out, err := cfg.EffectivePreset()
		exitOnError(err)
		fmt.Println(out)
	case "model":
		if len(args) < 2 {
			exitOnError(errors.New("cavekit config model: task type required"))
		}
		out, err := cfg.Model(args[1])
		exitOnError(err)
		fmt.Println(out)
	case "show":
		out, err := cfg.Show()
		exitOnError(err)
		fmt.Println(out)
	case "summary":
		out, err := cfg.SummaryLine()
		exitOnError(err)
		fmt.Println(out)
	case "presets":
		fmt.Println(configPresetsTable())
	case "caveman-active":
		if len(args) < 2 {
			exitOnError(errors.New("cavekit config caveman-active: phase required"))
		}
		active, err := cfg.CavemanActive(args[1])
		exitOnError(err)
		if active {
			fmt.Println("true")
		} else {
			fmt.Println("false")
		}
	case "help", "--help", "-h":
		fmt.Println(configUsage())
	default:
		exitOnError(fmt.Errorf("unknown cavekit config command: %s", args[0]))
	}
}

func configUsage() string {
	lines := []string{
		"Usage: cavekit config {init|get|set|preset|list|path|source|source-path|effective-preset|model|show|summary|presets|caveman-active}",
		"  init                              Create/backfill global and project config files",
		"  get <key> [default]               Read an effective config value",
		"  set <key> <val> [--global|--project]",
		"                                    Write a config value (project by default)",
		"  preset <name> [--global]          Convenience alias for setting bp_model_preset",
		"  list [--global|--project]         Show raw config key=value pairs from one file",
		"  path [--global|--project]         Print a config file path",
		"  source <key>                      Print value source: project | global | default",
		"  source-path <key>                 Print the path that supplied the value",
		"  effective-preset                  Print the effective model preset",
		"  model <task-type>                 Resolve model for reasoning | execution | exploration",
		"  show                              Print effective preset, source, and resolved models",
		"  summary                           Print a one-line preset summary",
		"  presets                           Print the built-in preset table",
		"  caveman-active <phase>            Check if caveman mode is active for a phase",
	}
	return strings.Join(lines, "\n")
}
