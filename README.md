# godata

Lightweight Go framework for small data pipelines. It focuses on simple ETL building blocks and a small orchestration layer for scheduling and running jobs.

## Setup

Requirements:
- Go 1.21+

Install:
- `go get github.com/janmarkuslanger/godata@latest`

Local development:
1) Clone the repo
2) Run `go test ./...`

## Packages

- `etl`: ETL pipelines (Read -> Step* -> Write)
- `scheduler`: schedule definitions (interval-based)
- `runner`: concurrent job execution with status tracking
- `orchestrator`: wiring of pipelines + schedules + persistence

## Design

- Small, focused packages so you can use only what you need.
- Steps are explicit; filters can drop records, hooks expose lifecycle events.
- Scheduling/execution are opt-in; nothing runs in the background unless you start it.
- File persistence is meant for a single instance; for multiple instances, use a shared database instead of local files.

## API Reference

### Package etl

Types:
- `Record`: `map[string]any` flowing through the pipeline.
- `Reader`: `func(ctx context.Context) ([]Record, error)` reads a batch.
- `Transform`: `func(ctx context.Context, record Record) (Record, error)` transforms a record.
- `Predicate`: `func(ctx context.Context, record Record) (bool, error)` decides if a record continues.
- `Writer`: `func(ctx context.Context, records []Record) error` writes a batch.

Pipeline:
- `New(name string) *Pipeline`: creates a pipeline with a name.
- `(*Pipeline).Name() string`: returns the name.
- `(*Pipeline).Read(reader Reader) *Pipeline`: sets the reader.
- `(*Pipeline).Transform(transform Transform) *Pipeline`: appends a transform step.
- `(*Pipeline).Filter(predicate Predicate) *Pipeline`: appends a filter step.
- `(*Pipeline).Write(writer Writer) *Pipeline`: sets the writer.
- `(*Pipeline).Hook(hook PipelineHook) *Pipeline`: sets pipeline hooks (nil uses `NoopPipelineHook`).
- `(*Pipeline).Run(ctx context.Context) error`: executes the pipeline.

Job:
- `NewJob(pipeline *Pipeline) (*Job, error)`: creates a job for a pipeline.
- `(*Job).Hook(hook JobHook) *Job`: sets job hooks (nil uses `NoopJobHook`).
- `(*Job).Run(ctx context.Context) error`: runs the job and stores timestamps/errors.
- `(*Job).Duration() time.Duration`: returns the runtime based on start/end.
- `Job` fields: `ID`, `Pipeline`, `Name`, `StartedAt`, `EndedAt`, `Err`.

Hooks:
- `PipelineInfo`: `PipelineName`.
- `PipelineHook`:
  - `OnPipelineStart(ctx, info)`
  - `OnPipelineEnd(ctx, info, err, dur)`
  - `OnReadStart(ctx, info)`
  - `OnReadEnd(ctx, info, records, err, dur)`
  - `OnStepStart(ctx, info, step)`
  - `OnStepEnd(ctx, info, step, err, dur)`
  - `OnWriteStart(ctx, info, records)`
  - `OnWriteEnd(ctx, info, err, dur)`
- `NoopPipelineHook`: empty implementation for embedding.
- `JobInfo`: `JobID`, `PipelineName`.
- `JobHook`:
  - `OnJobStart(ctx, info)`
  - `OnJobEnd(ctx, info, err, dur)`
- `NoopJobHook`: empty implementation for embedding.

Behavior:
- Steps run in order; filters may drop records and skip remaining steps.
- Errors are wrapped as `read failed: ...`, `step[N] failed: ...`, `write failed: ...`.

### Package scheduler

Types:
- `Schedule`: `Next(after time.Time) time.Time` decides the next run time.

Schedules:
- `Interval`: fixed-duration schedule.
- `Every(d time.Duration) Interval`: helper to build an interval schedule.

### Package runner

Errors:
- `ErrRunnerNil`, `ErrJobNil`, `ErrRunNotFound`.

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

Behavior:
- Runs are independent; overlapping is allowed by default.
- `Stop` cancels the run context.

### Package orchestrator

Errors:
- `ErrAlreadyRegistered`, `ErrUnknownPipeline`, `ErrSchedulerRunning`.

Orchestrator:
- `New(store, runner) *Orchestrator`: creates an orchestrator.
- `(*Orchestrator).Register(ctx, pipeline, schedule) error`: registers a pipeline.
- `(*Orchestrator).RunPipeline(ctx, name) (runner.Handle, error)`: runs a pipeline now.
- `(*Orchestrator).StartScheduler(ctx) error`: starts scheduling.
- `(*Orchestrator).StopScheduler()`: stops scheduling.

Persistence:
- `Store` interface: `UpsertPipeline`, `GetPipeline`, `ListPipelines`, `UpsertRun`, `GetRun`, `ListRuns`.
- `FileStore`: JSON-backed store without external deps.
- `NewFileStore(path string) *FileStore`: creates a file store.
- `PipelineRecord`: `Name`, `Enabled`, `UpdatedAt`.
- `RunRecord`: `ID`, `PipelineName`, `JobID`, `Status`, `StartedAt`, `EndedAt`, `Error`.
- `RunStatus`: `running`, `succeeded`, `failed`, `canceled`.

## Deployment

This library can run as a simple in-process pipeline or as a long-running service with scheduling.

Simple usage (no scheduler):
- You can run a single pipeline inside any Go process (CLI, batch job, or service).
- In this mode you only need `etl` (and optionally `runner`).

Always-on scheduling (internal scheduler):
- Run a small, long-lived service and let `orchestrator` trigger runs.
- This is the easiest way to keep all scheduling logic inside the app.

GCP (good fit for file-based persistence):
- Use Compute Engine with a small VM and Persistent Disk.
- Run the process via a systemd service (auto-start, restart on crash).
- Store `state.json` or SQLite on the persistent disk.
- Send logs to stdout/stderr and use Cloud Logging.

AWS (good fit for file-based persistence):
- Use EC2 with EBS for persistent storage.
- Run the process via a systemd service.
- Send logs to stdout/stderr and use CloudWatch.

Notes:
- JSON/SQLite is single-instance friendly. For multiple instances or HA, use a shared database instead of a local file.
- If you do not want an always-on process, use an external scheduler and only keep `runner` + `orchestrator`.

## Examples

### ETL pipeline

```go
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"strconv"

	"github.com/janmarkuslanger/godata/etl"
)

func main() {
	inputPath := "data/orders.csv"
	outputPath := "data/orders_paid.jsonl"

	p := etl.New("orders-cleanup").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			file, err := os.Open(inputPath)
			if err != nil {
				return nil, err
			}
			defer file.Close()

			reader := csv.NewReader(file)
			if _, err := reader.Read(); err != nil {
				return nil, err
			}

			var records []etl.Record
			for {
				row, err := reader.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					return nil, err
				}

				amount, err := strconv.ParseFloat(row[1], 64)
				if err != nil {
					return nil, err
				}

				records = append(records, etl.Record{
					"order_id": row[0],
					"amount":   amount,
					"status":   row[2],
				})
			}

			return records, nil
		}).
		Filter(func(ctx context.Context, record etl.Record) (bool, error) {
			status, _ := record["status"].(string)
			return status == "paid", nil
		}).
		Transform(func(ctx context.Context, record etl.Record) (etl.Record, error) {
			amount := record["amount"].(float64)
			record["gross"] = amount * 1.19
			return record, nil
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			out, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			defer out.Close()

			enc := json.NewEncoder(out)
			for _, record := range records {
				if err := enc.Encode(record); err != nil {
					return err
				}
			}
			return nil
		})

	_ = p.Run(context.Background())
}
```

### Hooks (embedding pattern)

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

```go
pipeline := etl.New("demo").
	Hook(&LoggingPipelineHook{})

job, _ := etl.NewJob(pipeline)
job.Hook(&LoggingJobHook{})
```

### Scheduler + orchestrator

```go
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/janmarkuslanger/godata/etl"
	"github.com/janmarkuslanger/godata/orchestrator"
	"github.com/janmarkuslanger/godata/scheduler"
)

func buildPipeline(inputPath, outputPath string) *etl.Pipeline {
	return etl.New("orders-cleanup").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			file, err := os.Open(inputPath)
			if err != nil {
				return nil, err
			}
			defer file.Close()

			reader := csv.NewReader(file)
			if _, err := reader.Read(); err != nil {
				return nil, err
			}

			var records []etl.Record
			for {
				row, err := reader.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					return nil, err
				}

				amount, err := strconv.ParseFloat(row[1], 64)
				if err != nil {
					return nil, err
				}

				records = append(records, etl.Record{
					"order_id": row[0],
					"amount":   amount,
					"status":   row[2],
				})
			}

			return records, nil
		}).
		Filter(func(ctx context.Context, record etl.Record) (bool, error) {
			status, _ := record["status"].(string)
			return status == "paid", nil
		}).
		Transform(func(ctx context.Context, record etl.Record) (etl.Record, error) {
			amount := record["amount"].(float64)
			record["gross"] = amount * 1.19
			return record, nil
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			out, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			defer out.Close()

			enc := json.NewEncoder(out)
			for _, record := range records {
				if err := enc.Encode(record); err != nil {
					return err
				}
			}
			return nil
		})
}

func main() {
	store := orchestrator.NewFileStore("state.json")
	orch := orchestrator.New(store, nil)

	pipeline := buildPipeline("data/orders.csv", "data/orders_paid.jsonl")
	_ = orch.Register(context.Background(), pipeline, scheduler.Every(5*time.Minute))
	_ = orch.StartScheduler(context.Background())
}
```

## Testing

Run all tests:
```
go test ./...
```
