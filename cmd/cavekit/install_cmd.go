package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const claudeMarketplaceName = "cavekit-local"

type claudeMarketplace struct {
	Name    string                    `json:"name"`
	Owner   map[string]string         `json:"owner"`
	Metadata map[string]string        `json:"metadata"`
	Plugins []claudeMarketplacePlugin `json:"plugins"`
}

type claudeMarketplacePlugin struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Source      string            `json:"source"`
	Author      map[string]string `json:"author"`
}

type claudeSettings struct {
	ExtraKnownMarketplaces map[string]claudeMarketplaceSetting `json:"extraKnownMarketplaces"`
	EnabledPlugins         map[string]bool                     `json:"enabledPlugins"`
}

type claudeMarketplaceSetting struct {
	Source claudeMarketplaceSource `json:"source"`
}

type claudeMarketplaceSource struct {
	Source string `json:"source"`
	Path   string `json:"path,omitempty"`
	Repo   string `json:"repo,omitempty"`
}

type claudePluginManifest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Plugins     []string `json:"plugins"`
}

func runInstall(args []string) {
	sourceDir, remaining, err := parseSourceDirFlag(args)
	exitOnError(err)
	for _, arg := range remaining {
		switch arg {
		case "--help", "-h":
			fmt.Println(installUsage())
			return
		default:
			exitOnError(fmt.Errorf("cavekit install: unknown argument %s", arg))
		}
	}

	sourceRoot := resolveSourceDir(sourceDir)
	homeDir := effectiveHomeDir()

	targetBinary, err := installBinaryPath()
	exitOnError(err)
	exitOnError(installBinary(targetBinary))
	exitOnError(configureClaude(sourceRoot, homeDir))
	exitOnError(syncCodexPlugin(sourceRoot, homeDir))

	if runtime.GOOS == "windows" && os.Getenv("CAVEKIT_INSTALL_SKIP_PATH") != "1" {
		exitOnError(addDirToUserPath(filepath.Dir(targetBinary)))
	}

	fmt.Println("Cavekit install complete.")
	fmt.Printf("Binary: %s\n", targetBinary)
	fmt.Println("Restart Claude Code and Codex if they are already running.")
}

func installBinaryPath() (string, error) {
	if dir := os.Getenv("CAVEKIT_INSTALL_BIN_DIR"); dir != "" {
		return filepath.Join(dir, binaryFileName()), nil
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(effectiveLocalAppData(), "Programs", "cavekit", "bin", binaryFileName()), nil
	}
	return filepath.Join("/usr/local/bin", binaryFileName()), nil
}

func binaryFileName() string {
	if runtime.GOOS == "windows" {
		return "cavekit.exe"
	}
	return "cavekit"
}

func installBinary(targetPath string) error {
	selfPath := os.Getenv("CAVEKIT_SELF_PATH")
	if selfPath == "" {
		var err error
		selfPath, err = os.Executable()
		if err != nil {
			return err
		}
	}
	if err := ensureDir(filepath.Dir(targetPath)); err != nil {
		return err
	}
	if err := copyFile(selfPath, targetPath); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		return os.Chmod(targetPath, 0o755)
	}
	return nil
}

func configureClaude(sourceRoot, homeDir string) error {
	claudeDir := filepath.Join(homeDir, ".claude")
	settingsPath := filepath.Join(claudeDir, "settings.json")
	marketplaceDir := filepath.Join(claudeDir, "plugins", "local", "cavekit-marketplace")
	pluginRoot := filepath.Join(marketplaceDir, ".claude-plugin")

	if err := ensureDir(pluginRoot); err != nil {
		return err
	}

	for _, alias := range []string{"ck", "bp"} {
		if err := createDirLink(sourceRoot, filepath.Join(marketplaceDir, alias)); err != nil {
			return err
		}
	}

	username := os.Getenv("USERNAME")
	if username == "" {
		username = os.Getenv("USER")
	}
	if username == "" {
		username = "local-user"
	}

	marketplace := claudeMarketplace{
		Name:  claudeMarketplaceName,
		Owner: map[string]string{"name": username},
		Metadata: map[string]string{
			"description": "Local Cavekit plugin marketplace",
			"version":     "2.0.0",
		},
		Plugins: []claudeMarketplacePlugin{
			{
				Name:        "ck",
				Description: "Cavekit framework with skills, commands, agents, and references",
				Version:     "2.0.0",
				Source:      "./ck",
				Author:      map[string]string{"name": username},
			},
			{
				Name:        "bp",
				Description: "[DEPRECATED - use /ck:* instead] Cavekit framework (legacy alias)",
				Version:     "2.0.0",
				Source:      "./bp",
				Author:      map[string]string{"name": username},
			},
		},
	}
	if err := writeJSONFile(filepath.Join(pluginRoot, "marketplace.json"), marketplace); err != nil {
		return err
	}

	manifest := claudePluginManifest{
		Name:        "cavekit-marketplace",
		Description: "Local Cavekit plugin marketplace",
		Version:     "2.0.0",
		Plugins:     []string{"ck", "bp"},
	}
	if err := writeJSONFile(filepath.Join(pluginRoot, "plugin.json"), manifest); err != nil {
		return err
	}

	settings := claudeSettings{
		ExtraKnownMarketplaces: map[string]claudeMarketplaceSetting{},
		EnabledPlugins:         map[string]bool{},
	}
	if fileExists(settingsPath) {
		raw, err := os.ReadFile(settingsPath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &settings); err != nil {
			return err
		}
	}
	if settings.ExtraKnownMarketplaces == nil {
		settings.ExtraKnownMarketplaces = map[string]claudeMarketplaceSetting{}
	}
	if settings.EnabledPlugins == nil {
		settings.EnabledPlugins = map[string]bool{}
	}
	settings.ExtraKnownMarketplaces[claudeMarketplaceName] = claudeMarketplaceSetting{
		Source: claudeMarketplaceSource{
			Source: "directory",
			Path:   marketplaceDir,
		},
	}
	settings.EnabledPlugins["ck@"+claudeMarketplaceName] = true
	settings.EnabledPlugins["bp@"+claudeMarketplaceName] = true

	return writeJSONFile(settingsPath, settings)
}

func installUsage() string {
	lines := []string{
		"Usage: cavekit install [--source-dir <path>]",
		"",
		"Install Cavekit natively for Claude Code and Codex.",
		"",
		"Options:",
		"  --source-dir <path>   Plugin source tree to link and sync",
	}
	return strings.Join(lines, "\n")
}
