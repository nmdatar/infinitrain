package scheduler

import "time"

// Now returns the current time - useful for testing and consistency
func Now() time.Time {
	return time.Now()
}
