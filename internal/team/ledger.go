package team

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

func ReadLedger(root string, stderr io.Writer) ([]LedgerEvent, error) {
	path := LedgerPath(root)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var events []LedgerEvent
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event LedgerEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			if stderr != nil {
				fmt.Fprintf(stderr, "warning: malformed ledger line %d: %v\n", lineNo, err)
			}
			continue
		}
		event.Owner = normalizeEmail(event.Owner)
		event.line = lineNo
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(events, func(i, j int) bool {
		ti := parseEventTime(events[i].TS)
		tj := parseEventTime(events[j].TS)
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		if events[i].Session != events[j].Session {
			return events[i].Session < events[j].Session
		}
		return events[i].line < events[j].line
	})

	return events, nil
}

func AppendLedgerEvent(root string, event LedgerEvent) error {
	if err := ensureDir(TeamDir(root)); err != nil {
		return err
	}
	file, err := os.OpenFile(LedgerPath(root), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func ActiveClaims(events []LedgerEvent, ttl time.Duration, now time.Time) map[string]ActiveClaim {
	type claimState struct {
		task          string
		owner         string
		host          string
		session       string
		acquiredAt    string
		lastHeartbeat string
		leaseUntil    string
		closed        bool
	}

	states := map[string]*claimState{}
	for _, event := range events {
		key := event.Task + "|" + event.Session
		switch event.Type {
		case EventClaim:
			states[key] = &claimState{
				task:          event.Task,
				owner:         event.Owner,
				host:          event.Host,
				session:       event.Session,
				acquiredAt:    event.TS,
				lastHeartbeat: event.TS,
				leaseUntil:    event.LeaseUntil,
			}
		case EventHeartbeat:
			state := states[key]
			if state == nil || state.closed {
				continue
			}
			state.lastHeartbeat = event.TS
			if event.LeaseUntil != "" {
				state.leaseUntil = event.LeaseUntil
			}
		case EventRelease, EventComplete:
			state := states[key]
			if state != nil {
				state.closed = true
			}
		}
	}

	active := map[string]ActiveClaim{}
	for _, state := range states {
		if state.closed || state.task == "" {
			continue
		}
		leaseUntil := parseEventTime(state.leaseUntil)
		lastHeartbeat := parseEventTime(state.lastHeartbeat)
		fresh := now.UTC().Before(leaseUntil) || now.UTC().Sub(lastHeartbeat) < ttl
		if !fresh {
			continue
		}
		claim := ActiveClaim{
			Task:          state.task,
			Owner:         state.owner,
			Host:          state.host,
			Session:       state.session,
			AcquiredAt:    state.acquiredAt,
			LastHeartbeat: state.lastHeartbeat,
			LeaseUntil:    state.leaseUntil,
			Stale:         false,
		}

		existing, ok := active[state.task]
		if !ok || activeClaimLess(claim, existing) {
			active[state.task] = claim
		}
	}
	return active
}

func CompletedTasks(events []LedgerEvent) map[string]bool {
	done := make(map[string]bool)
	for _, event := range events {
		if event.Type == EventComplete && event.Task != "" {
			done[event.Task] = true
		}
	}
	return done
}

func parseEventTime(value string) time.Time {
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return ts.UTC()
}

func activeClaimLess(left, right ActiveClaim) bool {
	leftTS := parseEventTime(left.AcquiredAt)
	rightTS := parseEventTime(right.AcquiredAt)
	if !leftTS.Equal(rightTS) {
		return leftTS.Before(rightTS)
	}
	return left.Session < right.Session
}
