package team

import (
	"os"
	"testing"
)

func TestDefaultConfig_Defaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.LeaseTTLSeconds != 30*60 {
		t.Fatalf("expected 30 min default lease TTL, got %d", cfg.LeaseTTLSeconds)
	}
	if cfg.HeartbeatIntervalSeconds != 60 {
		t.Fatalf("expected 60s heartbeat interval, got %d", cfg.HeartbeatIntervalSeconds)
	}
	if cfg.HeartbeatPublishEvery != 3 {
		t.Fatalf("expected publish-every 3, got %d", cfg.HeartbeatPublishEvery)
	}
	if cfg.AllowOffline {
		t.Fatal("expected AllowOffline default false — strict mode")
	}
}

func TestLoadConfig_FillsMissingPublishEvery(t *testing.T) {
	root := t.TempDir()
	if err := EnsureLedger(root); err != nil {
		t.Fatalf("ensure ledger: %v", err)
	}
	// Simulate a legacy config that predates heartbeat_publish_every.
	legacy := []byte(`{"lease_ttl_seconds": 600, "heartbeat_interval_seconds": 45}`)
	if err := os.WriteFile(ConfigPath(root), legacy, 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.HeartbeatPublishEvery != defaultHeartbeatPublishEvery {
		t.Fatalf("expected publish-every filled to default, got %d", cfg.HeartbeatPublishEvery)
	}
	if cfg.LeaseTTLSeconds != 600 {
		t.Fatalf("expected explicit lease TTL preserved, got %d", cfg.LeaseTTLSeconds)
	}
}
