package scheduler

import (
	"testing"
	"time"
)

func TestMultiNextEarliest(t *testing.T) {
	schedule := Multi{
		Schedules: []Schedule{
			DailyAt{
				Times:    []ClockTime{{Hour: 3, Minute: 0}},
				Location: time.UTC,
			},
			DailyAt{
				Times:    []ClockTime{{Hour: 7, Minute: 0}},
				Weekdays: []time.Weekday{time.Monday},
				Location: time.UTC,
			},
		},
	}

	after := time.Date(2024, time.January, 1, 3, 30, 0, 0, time.UTC) // Monday
	want := time.Date(2024, time.January, 1, 7, 0, 0, 0, time.UTC)

	if got := schedule.Next(after); !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestMultiSkipsZeroTimes(t *testing.T) {
	schedule := Multi{
		Schedules: []Schedule{
			DailyAt{},
			Every(2 * time.Hour),
		},
	}

	after := time.Date(2024, time.January, 1, 1, 0, 0, 0, time.UTC)
	want := after.Add(2 * time.Hour)

	if got := schedule.Next(after); !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
