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

- `etl`: simple ETL-style pipeline (Read -> Transform* -> Write)

## API

### Package etl

Core types:
- `Record`: `map[string]any` flowing through the pipeline.
- `Reader`: `func(ctx context.Context) ([]Record, error)` reads a batch.
- `Transform`: `func(ctx context.Context, record Record) (Record, error)` transforms a record.
- `Writer`: `func(ctx context.Context, records []Record) error` writes a batch.

Pipeline builder:
- `New(name string) *Pipeline`: creates a pipeline with a name.
- `(*Pipeline).Name() string`: returns the name.
- `(*Pipeline).Read(reader Reader) *Pipeline`: sets the reader.
- `(*Pipeline).Transform(transform Transform) *Pipeline`: appends a transform.
- `(*Pipeline).Write(writer Writer) *Pipeline`: sets the writer.
- `(*Pipeline).Run(ctx context.Context) error`: executes the pipeline.

Error behavior:
- read errors are wrapped as `read failed: ...`
- transform errors are wrapped as `transform[N] failed: ...`
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
		Write(func(ctx context.Context, records []etl.Record) error {
			fmt.Println(records)
			return nil
		})

	_ = p.Run(context.Background())
}
```

## Testing

Run all tests:
```
go test ./...
```
