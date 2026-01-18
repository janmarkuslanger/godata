package scheduler

import "time"

// Multi combines multiple schedules and returns the earliest next time.
type Multi struct {
	Schedules []Schedule
}

// Next returns the earliest next time across all schedules.
func (m Multi) Next(after time.Time) time.Time {
	var next time.Time
	for _, schedule := range m.Schedules {
		if schedule == nil {
			continue
		}
		candidate := schedule.Next(after)
		if candidate.IsZero() {
			continue
		}
		if next.IsZero() || candidate.Before(next) {
			next = candidate
		}
	}
	return next
}
