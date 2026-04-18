package team

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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
