package team

import (
	"path/filepath"
	"strings"
)

// PathsOverlap reports whether any claim-path pattern in a would cover a file
// claimed by b (or vice versa). Patterns follow extended glob semantics:
//   - `**` matches any number of path segments.
//   - `*` matches a single segment.
//   - Literal prefixes (e.g. `src/auth/`) match as directory prefixes.
//
// Two claims overlap when there exists at least one concrete path that either
// side's patterns could both match.
func PathsOverlap(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		// Empty path sets mean "unscoped" — unscoped claims block everything to
		// preserve legacy exclusive-task semantics.
		return true
	}
	for _, pa := range a {
		for _, pb := range b {
			if patternsIntersect(pa, pb) {
				return true
			}
		}
	}
	return false
}

// MatchAny reports whether path is covered by any of the patterns.
func MatchAny(patterns []string, path string) bool {
	path = filepath.ToSlash(path)
	for _, p := range patterns {
		if matchGlob(p, path) {
			return true
		}
	}
	return false
}

// patternsIntersect is a pragmatic test: if either pattern is a prefix of the
// other after trimming glob metacharacters, or one pattern matches the literal
// non-glob portion of the other, we consider them overlapping.
func patternsIntersect(a, b string) bool {
	a = filepath.ToSlash(strings.TrimSpace(a))
	b = filepath.ToSlash(strings.TrimSpace(b))
	if a == "" || b == "" {
		return true
	}
	if a == b {
		return true
	}
	litA := globLiteralPrefix(a)
	litB := globLiteralPrefix(b)
	if strings.HasPrefix(litA, litB) || strings.HasPrefix(litB, litA) {
		return true
	}
	if matchGlob(a, litB) || matchGlob(b, litA) {
		return true
	}
	return false
}

// matchGlob implements `**`-aware matching on top of filepath.Match.
func matchGlob(pattern, path string) bool {
	if pattern == path {
		return true
	}
	if !strings.Contains(pattern, "**") {
		ok, _ := filepath.Match(pattern, path)
		if ok {
			return true
		}
		// Tolerate directory-prefix patterns (e.g. "src/auth/" matches "src/auth/foo.go").
		if strings.HasSuffix(pattern, "/") && strings.HasPrefix(path, pattern) {
			return true
		}
		return false
	}
	return matchDoubleStar(pattern, path)
}

// matchDoubleStar is a simple recursive `**` matcher. Not exhaustive but
// covers the common cases cavekit generates ("src/auth/**", "**/foo.go").
func matchDoubleStar(pattern, path string) bool {
	parts := strings.Split(pattern, "/")
	segs := strings.Split(path, "/")
	return matchSegments(parts, segs)
}

func matchSegments(parts, segs []string) bool {
	for i, p := range parts {
		if p == "**" {
			if i == len(parts)-1 {
				return true
			}
			rest := parts[i+1:]
			for j := 0; j <= len(segs); j++ {
				if matchSegments(rest, segs[j:]) {
					return true
				}
			}
			return false
		}
		if len(segs) == 0 {
			return false
		}
		ok, _ := filepath.Match(p, segs[0])
		if !ok {
			return false
		}
		segs = segs[1:]
	}
	return len(segs) == 0
}

// globLiteralPrefix returns the longest prefix of a pattern before any glob
// metacharacter. Used as a coarse overlap heuristic.
func globLiteralPrefix(pattern string) string {
	for i, ch := range pattern {
		switch ch {
		case '*', '?', '[':
			return pattern[:i]
		}
	}
	return pattern
}
