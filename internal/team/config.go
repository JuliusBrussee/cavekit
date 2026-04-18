package team

import "os"

func LoadConfig(root string) (Config, error) {
	cfg := DefaultConfig()
	path := ConfigPath(root)
	if !fileExists(path) {
		return cfg, nil
	}
	if err := readJSON(path, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.LeaseTTLSeconds <= 0 {
		cfg.LeaseTTLSeconds = defaultLeaseTTLSeconds
	}
	if cfg.HeartbeatIntervalSeconds <= 0 {
		cfg.HeartbeatIntervalSeconds = defaultHeartbeatIntervalSeconds
	}
	if cfg.FetchIntervalSeconds < 0 {
		cfg.FetchIntervalSeconds = defaultFetchIntervalSeconds
	}
	return cfg, nil
}

func WriteDefaultConfig(root string) error {
	if err := ensureDir(TeamDir(root)); err != nil {
		return err
	}
	return writeJSON(ConfigPath(root), DefaultConfig())
}

func EnsureLedger(root string) error {
	if err := ensureDir(TeamDir(root)); err != nil {
		return err
	}
	if !fileExists(LedgerPath(root)) {
		if err := os.WriteFile(LedgerPath(root), nil, 0o644); err != nil {
			return err
		}
	}
	return ensureDir(LeasesDir(root))
}
