package team

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var taskIDPattern = regexp.MustCompile(`^T-([A-Za-z0-9]+-)*[A-Za-z0-9]+$`)

type Config struct {
	LeaseTTLSeconds         int `json:"lease_ttl_seconds"`
	HeartbeatIntervalSeconds int `json:"heartbeat_interval_seconds"`
	FetchIntervalSeconds    int `json:"fetch_interval_seconds,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		LeaseTTLSeconds:         defaultLeaseTTLSeconds,
		HeartbeatIntervalSeconds: defaultHeartbeatIntervalSeconds,
		FetchIntervalSeconds:    defaultFetchIntervalSeconds,
	}
}

type Identity struct {
	Email    string `json:"email"`
	Name     string `json:"name,omitempty"`
	Session  string `json:"session"`
	JoinedAt string `json:"joined_at"`
}

type Lease struct {
	Owner       string `json:"owner"`
	Host        string `json:"host"`
	PID         int    `json:"pid"`
	Session     string `json:"session"`
	AcquiredAt  string `json:"acquired_at"`
	HeartbeatAt string `json:"heartbeat_at"`
	ExpiresAt   string `json:"expires_at"`
}

type EventType string

const (
	EventClaim     EventType = "claim"
	EventRelease   EventType = "release"
	EventComplete  EventType = "complete"
	EventHeartbeat EventType = "heartbeat"
	EventNote      EventType = "note"
)

type LedgerEvent struct {
	TS         string    `json:"ts"`
	Type       EventType `json:"type"`
	Task       string    `json:"task"`
	Owner      string    `json:"owner"`
	Host       string    `json:"host"`
	Session    string    `json:"session"`
	LeaseUntil string    `json:"lease_until,omitempty"`
	Note       string    `json:"note,omitempty"`

	line int
}

type ActiveClaim struct {
	Task          string `json:"task"`
	Owner         string `json:"owner"`
	Host          string `json:"host"`
	Session       string `json:"session"`
	AcquiredAt    string `json:"acquired_at"`
	LastHeartbeat string `json:"last_heartbeat,omitempty"`
	LeaseUntil    string `json:"lease_until,omitempty"`
	Stale         bool   `json:"stale"`
}

type RosterMember struct {
	Identity string `json:"identity"`
}

type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	return e.Message
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func shortOwner(email string) string {
	local := email
	if idx := strings.IndexByte(email, '@'); idx > 0 {
		local = email[:idx]
	}
	if len(local) > 16 {
		return local[:16]
	}
	return local
}

func validateTaskID(taskID string) error {
	if !taskIDPattern.MatchString(taskID) {
		return fmt.Errorf("invalid task id: %s", taskID)
	}
	return nil
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	), nil
}
