package etl_test

import (
	"context"
	"errors"
	"github.com/janmarkuslanger/godata/etl"
	"reflect"
	"strings"
	"testing"
)

func TestPipelineName(t *testing.T) {
	pipeline := etl.New("demo")
	if pipeline.Name() != "demo" {
		t.Fatalf("expected name 'demo', got %q", pipeline.Name())
	}
}

func TestPipelineRunSuccess(t *testing.T) {
	ctx := context.Background()

	var got []etl.Record
	writerCalls := 0

	pipeline := etl.New("numbers").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			return []etl.Record{
				{"value": 1},
				{"value": 2},
			}, nil
		}).
		Transform(func(ctx context.Context, record etl.Record) (etl.Record, error) {
			record["value"] = record["value"].(int) + 1
			return record, nil
		}).
		Filter(func(ctx context.Context, record etl.Record) (bool, error) {
			return record["value"].(int)%2 == 0, nil
		}).
		Transform(func(ctx context.Context, record etl.Record) (etl.Record, error) {
			record["value"] = record["value"].(int) * 2
			return record, nil
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			writerCalls++
			got = append(got, records...)
			return nil
		})

	if err := pipeline.Run(ctx); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if writerCalls != 1 {
		t.Fatalf("expected writer to be called once, got %d", writerCalls)
	}

	want := []etl.Record{
		{"value": 4},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected records: %#v", got)
	}
}

func TestPipelineRunValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		build   func() *etl.Pipeline
		wantErr string
	}{
		{
			name: "nil pipeline",
			build: func() *etl.Pipeline {
				return nil
			},
			wantErr: "pipeline is nil",
		},
		{
			name: "missing name",
			build: func() *etl.Pipeline {
				return etl.New("").
					Read(func(ctx context.Context) ([]etl.Record, error) { return nil, nil }).
					Write(func(ctx context.Context, records []etl.Record) error { return nil })
			},
			wantErr: "pipeline name must not be empty",
		},
		{
			name: "missing reader",
			build: func() *etl.Pipeline {
				return etl.New("demo").
					Write(func(ctx context.Context, records []etl.Record) error { return nil })
			},
			wantErr: "reader must be set via Read",
		},
		{
			name: "missing writer",
			build: func() *etl.Pipeline {
				return etl.New("demo").
					Read(func(ctx context.Context) ([]etl.Record, error) { return nil, nil })
			},
			wantErr: "writer must be set via Write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := tt.build()
			err := pipeline.Run(ctx)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestPipelineRunErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("read error", func(t *testing.T) {
		readErr := errors.New("read broke")
		pipeline := etl.New("demo").
			Read(func(ctx context.Context) ([]etl.Record, error) { return nil, readErr }).
			Write(func(ctx context.Context, records []etl.Record) error { return nil })

		err := pipeline.Run(ctx)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, readErr) {
			t.Fatalf("expected wrapped error, got %v", err)
		}
		if !strings.Contains(err.Error(), "read failed") {
			t.Fatalf("expected read failed prefix, got %q", err.Error())
		}
	})

	t.Run("step error", func(t *testing.T) {
		stepErr := errors.New("transform broke")
		pipeline := etl.New("demo").
			Read(func(ctx context.Context) ([]etl.Record, error) { return []etl.Record{{"value": 1}}, nil }).
			Transform(func(ctx context.Context, record etl.Record) (etl.Record, error) { return nil, stepErr }).
			Write(func(ctx context.Context, records []etl.Record) error { return nil })

		err := pipeline.Run(ctx)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, stepErr) {
			t.Fatalf("expected wrapped error, got %v", err)
		}
		if !strings.Contains(err.Error(), "step[0] failed") {
			t.Fatalf("expected step prefix, got %q", err.Error())
		}
	})

	t.Run("write error", func(t *testing.T) {
		writeErr := errors.New("write broke")
		pipeline := etl.New("demo").
			Read(func(ctx context.Context) ([]etl.Record, error) { return []etl.Record{{"value": 1}}, nil }).
			Write(func(ctx context.Context, records []etl.Record) error { return writeErr })

		err := pipeline.Run(ctx)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, writeErr) {
			t.Fatalf("expected wrapped error, got %v", err)
		}
		if !strings.Contains(err.Error(), "write failed") {
			t.Fatalf("expected write failed prefix, got %q", err.Error())
		}
	})
}
