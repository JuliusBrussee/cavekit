package team

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

type LeaseCreateResult struct {
	Created  bool
	Existing *Lease
	Fresh    bool
}

func ReadLease(root, taskID string) (Lease, error) {
	var lease Lease
	if err := readJSON(LeasePath(root, taskID), &lease); err != nil {
		return Lease{}, err
	}
	return lease, nil
}

func IsLeaseFresh(lease Lease, now time.Time, ttl time.Duration) bool {
	expiresAt, err := time.Parse(time.RFC3339, lease.ExpiresAt)
	if err != nil {
		return false
	}
	heartbeatAt, err := time.Parse(time.RFC3339, lease.HeartbeatAt)
	if err != nil {
		return false
	}
	return now.UTC().Before(expiresAt.UTC()) && now.UTC().Sub(heartbeatAt.UTC()) < ttl
}

func TryCreateLease(root, taskID string, lease Lease, now time.Time, ttl time.Duration) (LeaseCreateResult, error) {
	if err := ensureDir(LeasesDir(root)); err != nil {
		return LeaseCreateResult{}, err
	}
	path := LeasePath(root, taskID)
	data, err := json.MarshalIndent(lease, "", "  ")
	if err != nil {
		return LeaseCreateResult{}, err
	}
	data = append(data, '\n')

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err == nil {
		defer file.Close()
		if _, writeErr := file.Write(data); writeErr != nil {
			return LeaseCreateResult{}, writeErr
		}
		return LeaseCreateResult{Created: true}, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return LeaseCreateResult{}, err
	}

	existing, readErr := ReadLease(root, taskID)
	if readErr != nil {
		return LeaseCreateResult{}, readErr
	}
	return LeaseCreateResult{
		Created:  false,
		Existing: &existing,
		Fresh:    IsLeaseFresh(existing, now, ttl),
	}, nil
}

func WriteLease(root, taskID string, lease Lease) error {
	return writeJSON(LeasePath(root, taskID), lease)
}

func DeleteLease(root, taskID string) error {
	err := os.Remove(LeasePath(root, taskID))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
