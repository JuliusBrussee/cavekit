package team

import (
	"testing"
	"time"
)

func TestTryCreateLease_DetectsFreshAndStale(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	ttl := 10 * time.Minute

	lease := Lease{
		Owner:       "dev@example.com",
		Host:        "host",
		PID:         123,
		Session:     "session-a",
		AcquiredAt:  now.Format(time.RFC3339),
		HeartbeatAt: now.Format(time.RFC3339),
		ExpiresAt:   now.Add(ttl).Format(time.RFC3339),
	}
	result, err := TryCreateLease(root, "T-001", lease, now, ttl)
	if err != nil {
		t.Fatalf("create lease: %v", err)
	}
	if !result.Created {
		t.Fatal("expected first lease creation to succeed")
	}

	result, err = TryCreateLease(root, "T-001", lease, now.Add(time.Minute), ttl)
	if err != nil {
		t.Fatalf("recreate lease: %v", err)
	}
	if result.Created || !result.Fresh {
		t.Fatalf("expected fresh collision, got %+v", result)
	}

	stale := lease
	stale.HeartbeatAt = now.Add(-20 * time.Minute).Format(time.RFC3339)
	stale.ExpiresAt = now.Add(-5 * time.Minute).Format(time.RFC3339)
	if err := WriteLease(root, "T-002", stale); err != nil {
		t.Fatalf("write stale lease: %v", err)
	}
	result, err = TryCreateLease(root, "T-002", lease, now, ttl)
	if err != nil {
		t.Fatalf("check stale lease: %v", err)
	}
	if result.Created || result.Fresh {
		t.Fatalf("expected stale collision, got %+v", result)
	}
}
