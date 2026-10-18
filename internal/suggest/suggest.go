// Package suggest finds the closest known name to a mistyped one, for the
// "did you mean" hints in config and theme errors.
package suggest

import (
	"slices"
	"strings"
)

// MaxDistance is the largest edit distance that still counts as a typo.
const MaxDistance = 2

// Closest returns the candidate nearest to unknown by edit distance, or ""
// when none is within MaxDistance. The comparison ignores case. Ties go to
// the candidate that sorts first, so the hint is the same on every run
// whatever order the candidates come in.
func Closest(unknown string, candidates []string) string {
	if unknown == "" {
		return ""
	}
	sorted := slices.Clone(candidates)
	slices.Sort(sorted)
	lower := strings.ToLower(unknown)
	best := ""
	bestDist := MaxDistance + 1
	for _, name := range sorted {
		if name == unknown {
			return ""
		}
		if d := Distance(lower, strings.ToLower(name)); d < bestDist {
			bestDist = d
			best = name
		}
	}
	return best
}

// Distance is the Levenshtein edit distance between a and b, by byte.
func Distance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}
