package scheduler

import "time"

// Schedule decides when the next run should occur after the given time.
type Schedule interface {
	Next(after time.Time) time.Time
}
