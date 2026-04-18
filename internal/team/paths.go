package team

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	Schema = "cavekit.team.v1"

	managedBlockStart = "# >>> cavekit-team"
	managedBlockEnd   = "# <<< cavekit-team"

	defaultLeaseTTLSeconds          = 30 * 60
	defaultHeartbeatIntervalSeconds = 60
	defaultFetchIntervalSeconds     = 30
	defaultHeartbeatPublishEvery    = 3
)

func TeamDir(root string) string {
	return filepath.Join(root, ".cavekit", "team")
}

func LedgerPath(root string) string {
	return filepath.Join(TeamDir(root), "ledger.jsonl")
}

func ConfigPath(root string) string {
	return filepath.Join(TeamDir(root), "config.json")
}

func IdentityPath(root string) string {
	return filepath.Join(TeamDir(root), "identity.json")
}

func LeasesDir(root string) string {
	return filepath.Join(TeamDir(root), "leases")
}

func LeasePath(root, taskID string) string {
	return filepath.Join(LeasesDir(root), taskID+".lock")
}

func RosterPath(root string) string {
	return filepath.Join(root, "context", "team", "roster.md")
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func atomicWrite(path string, content []byte) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	tmp := path + ".tmp." + time.Now().UTC().Format("20060102150405.000000000")
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicWrite(path, data)
}

func readJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsInitialized(root string) bool {
	return fileExists(LedgerPath(root)) || fileExists(ConfigPath(root))
}
