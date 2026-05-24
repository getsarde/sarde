package content

import "time"

// IsScheduled returns true if publishDate is non-zero and in the future.
func IsScheduled(publishDate time.Time, now time.Time) bool {
	if publishDate.IsZero() {
		return false
	}
	return publishDate.After(now)
}

// ShouldExclude returns true if a page should be excluded from output
// based on draft status and scheduling.
func ShouldExclude(draft bool, publishDate time.Time, includeDrafts, includeFuture bool, now time.Time) bool {
	if draft && !includeDrafts {
		return true
	}
	if IsScheduled(publishDate, now) && !includeFuture {
		return true
	}
	return false
}
