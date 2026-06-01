// pkg/utils/namematch.go
package utils

import (
	"strings"
	"unicode"
)

// NameMatchScore returns a confidence score (0.0 to 1.0) of how well two names match.
// It handles common Nigerian name variations: order differences, abbreviations, extra names.
func NameMatchScore(provided, resolved string) float64 {
	provided = normalizeName(provided)
	resolved = normalizeName(resolved)

	if provided == resolved {
		return 1.0
	}

	// Split into tokens
	provTokens := strings.Fields(provided)
	resTokens := strings.Fields(resolved)

	if len(provTokens) == 0 || len(resTokens) == 0 {
		return 0.0
	}

	// Count how many provided tokens appear in resolved tokens
	matches := 0
	for _, pt := range provTokens {
		for _, rt := range resTokens {
			if pt == rt || strings.HasPrefix(rt, pt) || strings.HasPrefix(pt, rt) {
				matches++
				break
			}
		}
	}

	// Score based on proportion of tokens matched
	maxTokens := len(provTokens)
	if len(resTokens) > maxTokens {
		maxTokens = len(resTokens)
	}

	return float64(matches) / float64(maxTokens)
}

// NamesMatch returns true if the names are sufficiently similar (threshold >= 0.6).
func NamesMatch(provided, resolved string) bool {
	return NameMatchScore(provided, resolved) >= 0.6
}

// normalizeName lowercases, removes non-alpha/space chars, and collapses whitespace.
func normalizeName(name string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) {
			b.WriteRune(r)
			prevSpace = false
		} else if unicode.IsSpace(r) || r == '-' {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}
