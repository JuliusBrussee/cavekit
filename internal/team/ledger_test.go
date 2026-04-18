package team

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestReadLedger_SkipsMalformedAndSorts(t *testing.T) {
	root := t.TempDir()
	if err := EnsureLedger(root); err != nil {
		t.Fatalf("ensure ledger: %v", err)
	}

	content := `{"ts":"2026-04-18T12:00:02Z","type":"claim","task":"T-002","owner":"b@example.com","host":"host","session":"bbb","lease_until":"2026-04-18T12:10:02Z"}
not-json
{"ts":"2026-04-18T12:00:01Z","type":"claim","task":"T-001","owner":"a@example.com","host":"host","session":"aaa","lease_until":"2026-04-18T12:10:01Z"}
`
	if err := os.WriteFile(LedgerPath(root), []byte(content), 0o644); err != nil {
		t.Fatalf("write ledger: %v", err)
	}

	var stderr bytes.Buffer
	events, err := ReadLedger(root, &stderr)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 parsed events, got %d", len(events))
	}
	if events[0].Task != "T-001" || events[1].Task != "T-002" {
		t.Fatalf("expected sorted events, got %+v", events)
	}
	if stderr.Len() == 0 {
		t.Fatal("expected malformed-line warning on stderr")
	}
}

func TestActiveClaims_FiltersReleasedAndStale(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 10, 0, 0, time.UTC)
	events := []LedgerEvent{
		{TS: "2026-04-18T12:00:00Z", Type: EventClaim, Task: "T-001", Owner: "a@example.com", Session: "aaa", LeaseUntil: "2026-04-18T12:20:00Z"},
		{TS: "2026-04-18T12:01:00Z", Type: EventRelease, Task: "T-001", Owner: "a@example.com", Session: "aaa"},
		{TS: "2026-04-18T12:02:00Z", Type: EventClaim, Task: "T-002", Owner: "b@example.com", Session: "bbb", LeaseUntil: "2026-04-18T12:20:00Z"},
		{TS: "2026-04-18T11:40:00Z", Type: EventClaim, Task: "T-003", Owner: "c@example.com", Session: "ccc", LeaseUntil: "2026-04-18T11:50:00Z"},
	}

	active := ActiveClaims(events, 10*time.Minute, now)
	if len(active) != 1 {
		t.Fatalf("expected 1 active claim, got %d (%+v)", len(active), active)
	}
	if active["T-002"].Owner != "b@example.com" {
		t.Fatalf("expected T-002 to remain active, got %+v", active["T-002"])
	}
}
