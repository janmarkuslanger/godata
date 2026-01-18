package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/janmarkuslanger/godata/etl"
	"github.com/janmarkuslanger/godata/orchestrator"
	"github.com/janmarkuslanger/godata/scheduler"
)

func buildPipeline() *etl.Pipeline {
	return etl.New("demo-log").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			return []etl.Record{
				{
					"message": "hello from the pipeline",
					"at":      time.Now().UTC(),
				},
			}, nil
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			for _, record := range records {
				log.Printf("demo record: %v", record)
			}
			return nil
		})
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pipeline := buildPipeline()
	orch := orchestrator.New(nil, nil)

	schedule := scheduler.DailyAt{
		Times: []scheduler.ClockTime{
			{Hour: 15, Minute: 15},
			{Hour: 17, Minute: 0},
		},
		Location: time.Local,
	}

	if err := orch.Register(ctx, pipeline, schedule); err != nil {
		log.Fatal(err)
	}
	if err := orch.StartScheduler(ctx); err != nil {
		log.Fatal(err)
	}

	log.Printf("scheduler running (15:15 and 17:00 local time)")
	<-ctx.Done()
}
