# godata

Lightweight Go framework for small data pipelines, organized into focused
packages. Public APIs and usage examples are listed below.

## Setup

Requirements:
- Go 1.21+

Local development:
1) Clone the repo
2) Run `go test ./...`

Using the module in another project:
1) `go get github.com/janmarkuslanger/godata@latest`

## Packages

- `etl`: simple ETL-style pipeline (Read -> Step* -> Write)
- `scheduler`: schedule definitions (e.g. fixed intervals).
- `runner`: concurrent job execution with status tracking.
- `orchestrator`: wiring of pipelines + schedules + persistence.

## API

### Package etl

Core types:
- `Record`: `map[string]any` flowing through the pipeline.
- `Reader`: `func(ctx context.Context) ([]Record, error)` reads a batch.
- `Transform`: `func(ctx context.Context, record Record) (Record, error)` transforms a record.
- `Predicate`: `func(ctx context.Context, record Record) (bool, error)` decides if a record continues.
- `Writer`: `func(ctx context.Context, records []Record) error` writes a batch.

Pipeline builder:
- `New(name string) *Pipeline`: creates a pipeline with a name.
- `(*Pipeline).Name() string`: returns the name.
- `(*Pipeline).Read(reader Reader) *Pipeline`: sets the reader.
- `(*Pipeline).Transform(transform Transform) *Pipeline`: appends a transform.
- `(*Pipeline).Filter(predicate Predicate) *Pipeline`: appends a filter.
- `(*Pipeline).Write(writer Writer) *Pipeline`: sets the writer.
- `(*Pipeline).Hook(hook PipelineHook) *Pipeline`: sets pipeline hooks (nil uses Noop).
- `(*Pipeline).Run(ctx context.Context) error`: executes the pipeline.

Step behavior:
- steps run in order; filters may drop records and skip remaining steps.

Jobs:
- `NewJob(pipeline *Pipeline) (*Job, error)`: creates a job for a pipeline.
- `(*Job).Hook(hook JobHook) *Job`: sets job hooks (nil uses Noop).
- `(*Job).Run(ctx context.Context) error`: runs the job and stores timestamps/errors.
- `(*Job).Duration() time.Duration`: returns the runtime based on start/end.
- `Job` fields: `ID`, `Name`, `StartedAt`, `EndedAt`, `Err`.

Hooks:
- `PipelineHook`: lifecycle events for pipeline, read/step/write stages.
- `JobHook`: lifecycle events for job start/end.
- `NoopPipelineHook` and `NoopJobHook` are provided for embedding.

Error behavior:
- read errors are wrapped as `read failed: ...`
- step errors are wrapped as `step[N] failed: ...`
- write errors are wrapped as `write failed: ...`

### Package scheduler

Core types:
- `Schedule`: `Next(after time.Time) time.Time` decides the next run time.

Schedules:
- `Interval`: fixed-duration schedule.
- `Every(d time.Duration) Interval`: helper to build an interval schedule.

### Package runner

Runner:
- `New() *Runner`: creates a runner.
- `(*Runner).Start(ctx, job) (Handle, error)`: runs a job asynchronously.
- `(*Runner).Stop(id string) error`: cancels a running job.
- `(*Runner).Active() []Result`: returns running jobs.
- `(*Runner).Result(id string) (Result, bool)`: returns the latest known result.

Run types:
- `Status`: `running`, `succeeded`, `failed`, `canceled`.
- `Result`: `ID`, `JobID`, `JobName`, `Status`, `StartedAt`, `EndedAt`, `Err`.
- `Handle`: `ID`, `JobID`, `JobName`, `StartedAt`, `Done()`, `Result()`.

Runner behavior:
- runs are independent; overlapping is allowed by default.

### Package orchestrator

Orchestrator:
- `New(store, runner) *Orchestrator`: creates an orchestrator.
- `(*Orchestrator).Register(ctx, pipeline, schedule) error`: registers a pipeline.
- `(*Orchestrator).RunPipeline(ctx, name) (runner.Handle, error)`: runs a pipeline now.
- `(*Orchestrator).StartScheduler(ctx) error`: starts scheduling.
- `(*Orchestrator).StopScheduler()`: stops scheduling.
- `(*Orchestrator).SetPipelineEnabled(ctx, name, enabled) error`: toggle scheduling.

Persistence:
- `Store` interface persists pipelines and runs.
- `FileStore`: JSON-backed store without external deps.
- `PipelineRecord`: `Name`, `Enabled`, `UpdatedAt`.
- `RunRecord`: `ID`, `PipelineName`, `JobID`, `Status`, `StartedAt`, `EndedAt`, `Error`.

## Example (package etl)

```go
package main

import (
	"context"
	"fmt"

	"github.com/janmarkuslanger/godata/etl"
)

func main() {
	p := etl.New("numbers").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			return []etl.Record{
				{"value": 1},
				{"value": 2},
			}, nil
		}).
		Transform(func(ctx context.Context, record etl.Record) (etl.Record, error) {
			record["value"] = record["value"].(int) * 2
			return record, nil
		}).
		Filter(func(ctx context.Context, record etl.Record) (bool, error) {
			return record["value"].(int) > 2, nil
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			fmt.Println(records)
			return nil
		})

	_ = p.Run(context.Background())
}
```

## Example (scheduler + runner + orchestrator)

```go
package main

import (
	"context"
	"time"

	"github.com/janmarkuslanger/godata/etl"
	"github.com/janmarkuslanger/godata/orchestrator"
	"github.com/janmarkuslanger/godata/scheduler"
)

func main() {
	store := orchestrator.NewFileStore("state.json")
	orch := orchestrator.New(store, nil)

	pipeline := etl.New("demo").
		Read(func(ctx context.Context) ([]etl.Record, error) { return nil, nil }).
		Write(func(ctx context.Context, records []etl.Record) error { return nil })

	_ = orch.Register(context.Background(), pipeline, scheduler.Every(5*time.Minute))
	_ = orch.StartScheduler(context.Background())
}
```

## Hooks (embedding pattern)

```go
type LoggingPipelineHook struct {
	etl.NoopPipelineHook
}

func (LoggingPipelineHook) OnPipelineEnd(ctx context.Context, info etl.PipelineInfo, err error, dur time.Duration) {
	// log or metrics here
}

type LoggingJobHook struct {
	etl.NoopJobHook
}

func (LoggingJobHook) OnJobEnd(ctx context.Context, info etl.JobInfo, err error, dur time.Duration) {
	// log or metrics here
}
```

Use them like:

```go
pipeline := etl.New("demo").
	Hook(&LoggingPipelineHook{})

job, _ := etl.NewJob(pipeline)
job.Hook(&LoggingJobHook{})
```

## Testing

Run all tests:
```
go test ./...
```
