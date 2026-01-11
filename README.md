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
