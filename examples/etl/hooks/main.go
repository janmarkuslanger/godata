package main

import (
	"context"
	"log"
	"time"

	"github.com/janmarkuslanger/godata/etl"
)

type PipelineLogger struct {
	etl.NoopPipelineHook
}

func (PipelineLogger) OnPipelineEnd(ctx context.Context, info etl.PipelineInfo, err error, dur time.Duration) {
	log.Printf("[%s] pipeline done in %s (err=%v)", info.PipelineName, dur, err)
}

type JobLogger struct {
	etl.NoopJobHook
}

func (JobLogger) OnJobEnd(ctx context.Context, info etl.JobInfo, err error, dur time.Duration) {
	log.Printf("[%s] job done in %s (err=%v)", info.PipelineName, dur, err)
}

func main() {
	ctx := context.Background()

	pipeline := etl.New("hooks").
		Hook(PipelineLogger{}).
		Read(func(ctx context.Context) ([]etl.Record, error) {
			return []etl.Record{
				{"value": 1},
			}, nil
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			log.Printf("wrote %d record(s)", len(records))
			return nil
		})

	job, err := etl.NewJob(pipeline)
	if err != nil {
		log.Fatal(err)
	}
	job.Hook(JobLogger{})

	if err := job.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
