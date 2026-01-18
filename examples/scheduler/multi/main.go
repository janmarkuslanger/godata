package main

import (
	"log"
	"time"

	"github.com/janmarkuslanger/godata/scheduler"
)

func main() {
	schedule := scheduler.Multi{
		Schedules: []scheduler.Schedule{
			scheduler.DailyAt{
				Times: []scheduler.ClockTime{
					{Hour: 3, Minute: 0},
				},
				Location: time.Local,
			},
			scheduler.DailyAt{
				Times: []scheduler.ClockTime{
					{Hour: 7, Minute: 0},
				},
				Weekdays: []time.Weekday{time.Monday},
				Location: time.Local,
			},
		},
	}

	after := time.Now()
	for i := 0; i < 5; i++ {
		next := schedule.Next(after)
		log.Printf("next run: %s", next.Format(time.RFC3339))
		after = next
	}
}
