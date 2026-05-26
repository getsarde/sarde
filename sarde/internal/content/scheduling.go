package content

import "time"

// IsScheduled returns true if publishDate is non-zero and in the future.
func IsScheduled(publishDate time.Time, now time.Time) bool {
	if publishDate.IsZero() {
		return false
	}
	return publishDate.After(now)
}

// IsExpired returns true if expiryDate is non-zero and not in the future.
func IsExpired(expiryDate time.Time, now time.Time) bool {
	if expiryDate.IsZero() {
		return false
	}
	return !expiryDate.After(now)
}

// ShouldExclude returns true if a page should be excluded from output
// based on draft status, scheduling, and expiry.
func ShouldExclude(draft bool, publishDate, expiryDate time.Time, includeDrafts, includeFuture, includeExpired bool, now time.Time) bool {
	if draft && !includeDrafts {
		return true
	}
	if IsScheduled(publishDate, now) && !includeFuture {
		return true
	}
	if IsExpired(expiryDate, now) && !includeExpired {
		return true
	}
	return false
}
