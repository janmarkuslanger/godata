package scheduler

import (
	"testing"
	"time"
)

func TestDailyAtNextSameDay(t *testing.T) {
	schedule := DailyAt{
		Times: []ClockTime{
			{Hour: 14, Minute: 30},
			{Hour: 2, Minute: 0},
		},
		Location: time.UTC,
	}

	after := time.Date(2024, time.January, 1, 1, 0, 0, 0, time.UTC)
	want := time.Date(2024, time.January, 1, 2, 0, 0, 0, time.UTC)

	if got := schedule.Next(after); !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestDailyAtNextLaterSameDay(t *testing.T) {
	schedule := DailyAt{
		Times:    []ClockTime{{Hour: 2, Minute: 0}, {Hour: 14, Minute: 30}},
		Location: time.UTC,
	}

	after := time.Date(2024, time.January, 1, 2, 0, 0, 0, time.UTC)
	want := time.Date(2024, time.January, 1, 14, 30, 0, 0, time.UTC)

	if got := schedule.Next(after); !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestDailyAtNextNextDay(t *testing.T) {
	schedule := DailyAt{
		Times:    []ClockTime{{Hour: 2, Minute: 0}},
		Location: time.UTC,
	}

	after := time.Date(2024, time.January, 1, 23, 0, 0, 0, time.UTC)
	want := time.Date(2024, time.January, 2, 2, 0, 0, 0, time.UTC)

	if got := schedule.Next(after); !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestDailyAtWeekdayFilter(t *testing.T) {
	schedule := DailyAt{
		Times:    []ClockTime{{Hour: 2, Minute: 0}},
		Weekdays: []time.Weekday{time.Monday},
		Location: time.UTC,
	}

	after := time.Date(2024, time.January, 1, 3, 0, 0, 0, time.UTC) // Monday
	want := time.Date(2024, time.January, 8, 2, 0, 0, 0, time.UTC)

	if got := schedule.Next(after); !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestDailyAtInvalidTimes(t *testing.T) {
	schedule := DailyAt{
		Times:    []ClockTime{{Hour: 25, Minute: 0}},
		Location: time.UTC,
	}

	if got := schedule.Next(time.Now()); !got.IsZero() {
		t.Fatalf("expected zero time, got %v", got)
	}
}
