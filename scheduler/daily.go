package scheduler

import (
	"sort"
	"time"
)

// ClockTime describes a time of day in 24h format.
type ClockTime struct {
	Hour   int
	Minute int
}

// DailyAt schedules runs at specific times each day.
// If Weekdays is set, runs only on those days.
type DailyAt struct {
	Times    []ClockTime
	Weekdays []time.Weekday
	Location *time.Location
}

// Next returns the next run time after the given time.
func (d DailyAt) Next(after time.Time) time.Time {
	times := normalizeTimes(d.Times)
	if len(times) == 0 {
		return time.Time{}
	}

	loc := d.Location
	if loc == nil {
		loc = time.Local
	}

	allowedDays := make(map[time.Weekday]bool, len(d.Weekdays))
	if len(d.Weekdays) > 0 {
		for _, day := range d.Weekdays {
			allowedDays[day] = true
		}
	}

	anchor := after.In(loc)
	for dayOffset := 0; dayOffset < 366; dayOffset++ {
		day := anchor.AddDate(0, 0, dayOffset)
		if len(allowedDays) > 0 && !allowedDays[day.Weekday()] {
			continue
		}

		for _, t := range times {
			candidate := time.Date(day.Year(), day.Month(), day.Day(), t.Hour, t.Minute, 0, 0, loc)
			if candidate.After(anchor) {
				return candidate
			}
		}
	}

	return time.Time{}
}

func normalizeTimes(times []ClockTime) []ClockTime {
	valid := make([]ClockTime, 0, len(times))
	for _, t := range times {
		if t.Hour < 0 || t.Hour > 23 || t.Minute < 0 || t.Minute > 59 {
			continue
		}
		valid = append(valid, t)
	}

	sort.Slice(valid, func(i, j int) bool {
		if valid[i].Hour == valid[j].Hour {
			return valid[i].Minute < valid[j].Minute
		}
		return valid[i].Hour < valid[j].Hour
	})

	return valid
}
