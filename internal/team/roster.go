package team

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// rosterFocus returns the comma-separated "Focus:" tokens for a given email
// in context/team/roster.md. Used by the scheduler to prefer tasks that match
// the identity's declared focus areas.
func rosterFocus(root, email string) []string {
	data, err := os.ReadFile(RosterPath(root))
	if err != nil {
		return nil
	}
	email = strings.ToLower(strings.TrimSpace(email))
	content := string(data)
	lines := strings.Split(content, "\n")
	var focus []string
	var currentFocus []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		switch {
		case strings.HasPrefix(lower, "focus:"):
			raw := strings.TrimSpace(trimmed[len("focus:"):])
			currentFocus = splitTrim(raw, ",")
		case strings.HasPrefix(lower, "identity:"):
			id := strings.ToLower(strings.TrimSpace(trimmed[len("Identity:"):]))
			if id == email {
				focus = append(focus, currentFocus...)
			}
			currentFocus = nil
		case strings.HasPrefix(lower, "### "):
			// New member section resets accumulator.
			currentFocus = nil
		}
	}
	return focus
}

func splitTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// containsFold is a case-insensitive strings.Contains.
func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

const rosterTemplate = `# Cavekit Team

## Members

### Example Member
Focus: backend, build-system
Personal bests: API schema changes, test harness cleanup
Identity: dev@example.com

`

func EnsureRoster(root string) (bool, error) {
	path := RosterPath(root)
	if fileExists(path) {
		return false, nil
	}
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(rosterTemplate), 0o644)
}

func ReadRosterMembers(root string) ([]RosterMember, error) {
	data, err := os.ReadFile(RosterPath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	re := regexp.MustCompile(`(?im)^\s*Identity:\s*(.+?)\s*$`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	members := make([]RosterMember, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		identity := normalizeEmail(strings.TrimSpace(match[1]))
		if identity == "" || seen[identity] {
			continue
		}
		seen[identity] = true
		members = append(members, RosterMember{Identity: identity})
	}
	return members, nil
}
