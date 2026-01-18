package main

import (
	"context"
	"log"
	"time"

	"github.com/janmarkuslanger/godata/etl"
	"github.com/janmarkuslanger/godata/runner"
)

func buildPipeline() *etl.Pipeline {
	return etl.New("concurrent").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			time.Sleep(100 * time.Millisecond)
			return []etl.Record{
				{"value": 1},
			}, nil
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
}

func main() {
	ctx := context.Background()
	r := runner.New()
	pipeline := buildPipeline()

	const runs = 3
	handles := make([]runner.Handle, 0, runs)

	for i := 0; i < runs; i++ {
		job, err := etl.NewJob(pipeline)
		if err != nil {
			log.Fatal(err)
		}

		handle, err := r.Start(ctx, job)
		if err != nil {
			log.Fatal(err)
		}
		handles = append(handles, handle)
	}

	for _, handle := range handles {
		<-handle.Done()
		if result, ok := r.Result(handle.ID); ok {
			log.Printf("run=%s status=%s err=%v", result.ID, result.Status, result.Err)
		}
	}
}
