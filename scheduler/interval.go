package scheduler

import "time"

// Interval schedules runs at a fixed duration.
type Interval struct {
	Every time.Duration
}

// Every creates an Interval schedule.
func Every(d time.Duration) Interval {
	return Interval{Every: d}
}

// Next returns the next run time after the given time.
func (i Interval) Next(after time.Time) time.Time {
	if i.Every <= 0 {
		return time.Time{}
	}
	return after.Add(i.Every)
}
