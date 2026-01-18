package main

import (
	"log"
	"time"

	"github.com/janmarkuslanger/godata/scheduler"
)

func main() {
	schedule := scheduler.DailyAt{
		Times: []scheduler.ClockTime{
			{Hour: 9, Minute: 0},
			{Hour: 17, Minute: 0},
		},
		Location: time.Local,
	}

	after := time.Now()
	for i := 0; i < 5; i++ {
		next := schedule.Next(after)
		log.Printf("next run: %s", next.Format(time.RFC3339))
		after = next
	}
}
