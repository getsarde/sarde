package workers

import (
	"runtime"

	"golang.org/x/sync/errgroup"
)

// Count returns the default bounded worker count for build work.
func Count() int {
	n := runtime.GOMAXPROCS(0)
	if n < 1 {
		return 1
	}
	return n
}

// Limit returns Count capped to the amount of available work.
func Limit(workItems int) int {
	n := Count()
	if workItems > 0 && workItems < n {
		return workItems
	}
	return n
}

// IOLimit returns a concurrency cap for I/O-bound file operations.
// SSD-backed filesystems benefit from more outstanding requests than
// there are CPU cores.
func IOLimit(workItems int) int {
	n := Count() * 2
	if workItems > 0 && workItems < n {
		return workItems
	}
	return n
}

// ShouldParallelize reports whether a build phase has enough work to benefit
// from worker fan-out. Small phases stay serial to avoid goroutine overhead.
func ShouldParallelize(enabled bool, workItems, workerCount int) bool {
	if !enabled || workItems < 2 {
		return false
	}
	if workerCount < 1 {
		workerCount = Count()
	}
	threshold := workerCount * 8
	if threshold < 16 {
		threshold = 16
	}
	return workItems >= threshold
}

// ParallelFor runs fn over every item, fanning out to workers when the work
// justifies it (per ShouldParallelize) and running serially otherwise. The
// serial path stops at the first error; the parallel path waits for started
// goroutines and returns the first error. workerCount caps parallel fan-out;
// when it is <= 0 the cap defaults to Limit(len(items)).
func ParallelFor[T any](items []T, parallel bool, workerCount int, fn func(i int, item T) error) error {
	if !ShouldParallelize(parallel, len(items), workerCount) {
		for i, item := range items {
			if err := fn(i, item); err != nil {
				return err
			}
		}
		return nil
	}

	limit := workerCount
	if limit <= 0 {
		limit = Limit(len(items))
	}
	g := new(errgroup.Group)
	g.SetLimit(limit)
	for i, item := range items {
		i, item := i, item
		g.Go(func() error {
			return fn(i, item)
		})
	}
	return g.Wait()
}
