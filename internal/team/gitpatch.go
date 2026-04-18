package team

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnsureGitignoreBlock ensures the managed block ignores every piece of the
// team state that is per-checkout (identity, leases, local cache, outbox, head
// pointer). Since the ledger now lives on its own ref branch, no team state is
// ever committed to the working branch.
func EnsureGitignoreBlock(root string, force bool) error {
	block := []string{
		managedBlockStart,
		".cavekit/team/leases/",
		".cavekit/team/identity.json",
		".cavekit/team/ledger.jsonl",
		".cavekit/team/ledger.head",
		".cavekit/team/outbox.jsonl",
		managedBlockEnd,
	}
	updated, err := upsertManagedBlock(filepath.Join(root, ".gitignore"), block, force)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(root, ".gitignore"), []byte(updated))
}

// EnsureGitattributesBlock used to install `merge=union` for the ledger. That
// is no longer required because the ledger isn't tracked on the working
// branch. The block is rewritten to empty so legacy installs drop the
// attribute cleanly. We still scan for stale entries and warn.
func EnsureGitattributesBlock(root string, force bool) ([]string, error) {
	path := filepath.Join(root, ".gitattributes")
	warnings := findConflictingAttributes(path)
	block := []string{
		managedBlockStart,
		"# cavekit team ledger is tracked via refs/heads/cavekit/team, not main",
		managedBlockEnd,
	}
	updated, err := upsertManagedBlock(path, block, force)
	if err != nil {
		return warnings, err
	}
	return warnings, atomicWrite(path, []byte(updated))
}

func upsertManagedBlock(path string, block []string, force bool) (string, error) {
	existing := ""
	if data, err := os.ReadFile(path); err == nil {
		existing = string(data)
	} else if !os.IsNotExist(err) {
		return "", err
	}

	lines := []string{}
	if existing != "" {
		lines = strings.Split(strings.ReplaceAll(existing, "\r\n", "\n"), "\n")
	}

	start, end := -1, -1
	for i, line := range lines {
		if line == managedBlockStart {
			start = i
		}
		if line == managedBlockEnd && start >= 0 {
			end = i
			break
		}
	}

	blockText := strings.Join(block, "\n")
	if start >= 0 && end >= start {
		prefix := append([]string{}, lines[:start]...)
		suffix := []string{}
		if end+1 < len(lines) {
			suffix = append(suffix, lines[end+1:]...)
		}
		combined := append(prefix, strings.Split(blockText, "\n")...)
		combined = append(combined, trimTrailingEmpty(suffix)...)
		return strings.Join(trimTrailingEmpty(combined), "\n") + "\n", nil
	}

	if existing == "" {
		return blockText + "\n", nil
	}

	if !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	return existing + blockText + "\n", nil
}

func trimTrailingEmpty(lines []string) []string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func findConflictingAttributes(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var warnings []string
	scanner := bufio.NewScanner(file)
	insideManaged := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch line {
		case managedBlockStart:
			insideManaged = true
			continue
		case managedBlockEnd:
			insideManaged = false
			continue
		}
		if insideManaged || line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, ".cavekit/team/ledger.jsonl") && !strings.Contains(line, "merge=union") {
			warnings = append(warnings, fmt.Sprintf("conflicting .gitattributes entry outside cavekit-team block: %s", line))
		}
	}
	return warnings
}
