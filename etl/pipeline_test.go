package etl

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestPipelineName(t *testing.T) {
	pipeline := New("demo")
	if pipeline.Name() != "demo" {
		t.Fatalf("expected name 'demo', got %q", pipeline.Name())
	}
}

func TestPipelineRunSuccess(t *testing.T) {
	ctx := context.Background()

	var got []Record
	writerCalls := 0

	pipeline := New("numbers").
		Read(func(ctx context.Context) ([]Record, error) {
			return []Record{
				{"value": 1},
				{"value": 2},
			}, nil
		}).
		Transform(func(ctx context.Context, record Record) (Record, error) {
			record["value"] = record["value"].(int) + 1
			return record, nil
		}).
		Transform(func(ctx context.Context, record Record) (Record, error) {
			record["value"] = record["value"].(int) * 2
			return record, nil
		}).
		Write(func(ctx context.Context, records []Record) error {
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

	want := []Record{
		{"value": 4},
		{"value": 6},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected records: %#v", got)
	}
}

func TestPipelineRunValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		build   func() *Pipeline
		wantErr string
	}{
		{
			name: "nil pipeline",
			build: func() *Pipeline {
				return nil
			},
			wantErr: "pipeline is nil",
		},
		{
			name: "missing name",
			build: func() *Pipeline {
				return New("").
					Read(func(ctx context.Context) ([]Record, error) { return nil, nil }).
					Write(func(ctx context.Context, records []Record) error { return nil })
			},
			wantErr: "pipeline name must not be empty",
		},
		{
			name: "missing reader",
			build: func() *Pipeline {
				return New("demo").
					Write(func(ctx context.Context, records []Record) error { return nil })
			},
			wantErr: "reader must be set via Read",
		},
		{
			name: "missing writer",
			build: func() *Pipeline {
				return New("demo").
					Read(func(ctx context.Context) ([]Record, error) { return nil, nil })
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
		pipeline := New("demo").
			Read(func(ctx context.Context) ([]Record, error) { return nil, readErr }).
			Write(func(ctx context.Context, records []Record) error { return nil })

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

	t.Run("transform error", func(t *testing.T) {
		transformErr := errors.New("transform broke")
		pipeline := New("demo").
			Read(func(ctx context.Context) ([]Record, error) { return []Record{{"value": 1}}, nil }).
			Transform(func(ctx context.Context, record Record) (Record, error) { return nil, transformErr }).
			Write(func(ctx context.Context, records []Record) error { return nil })

		err := pipeline.Run(ctx)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, transformErr) {
			t.Fatalf("expected wrapped error, got %v", err)
		}
		if !strings.Contains(err.Error(), "transform[0] failed") {
			t.Fatalf("expected transform prefix, got %q", err.Error())
		}
	})

	t.Run("write error", func(t *testing.T) {
		writeErr := errors.New("write broke")
		pipeline := New("demo").
			Read(func(ctx context.Context) ([]Record, error) { return []Record{{"value": 1}}, nil }).
			Write(func(ctx context.Context, records []Record) error { return writeErr })

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
