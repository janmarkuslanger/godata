package main

import (
	"context"
	"log"
	"time"

	"github.com/janmarkuslanger/godata/etl"
	"github.com/janmarkuslanger/godata/orchestrator"
	"github.com/janmarkuslanger/godata/runner"
)

func buildPipeline() *etl.Pipeline {
	return etl.New("cron-trigger").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			return []etl.Record{
				{"triggered_at": time.Now().UTC()},
			}, nil
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			log.Printf("cron run wrote %d record(s)", len(records))
			return nil
		})
}

func main() {
	ctx := context.Background()

	store := orchestrator.NewFileStore("state.json")
	r := runner.New()
	orch := orchestrator.New(store, r)

	pipeline := buildPipeline()
	if err := orch.Register(ctx, pipeline, nil); err != nil {
		log.Fatal(err)
	}

	handle, err := orch.RunPipeline(ctx, pipeline.Name())
	if err != nil {
		log.Fatal(err)
	}

	<-handle.Done()
	if result, ok := r.Result(handle.ID); ok {
		log.Printf("cron run status=%s err=%v", result.Status, result.Err)
	}
}
