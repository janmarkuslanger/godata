package etl_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/janmarkuslanger/godata/etl"
)

type pipelineHookRecorder struct {
	calls            []string
	info             etl.PipelineInfo
	readRecords      int
	readErr          error
	transformSteps   []int
	transformEndErrs []error
	writeRecords     int
	writeErr         error
	pipelineErr      error
	negativeDuration bool
}

func (h *pipelineHookRecorder) OnPipelineStart(ctx context.Context, info etl.PipelineInfo) {
	h.calls = append(h.calls, "pipeline_start")
	h.info = info
}

func (h *pipelineHookRecorder) OnPipelineEnd(ctx context.Context, info etl.PipelineInfo, err error, dur time.Duration) {
	h.calls = append(h.calls, "pipeline_end")
	h.pipelineErr = err
	h.checkDuration(dur)
}

func (h *pipelineHookRecorder) OnReadStart(ctx context.Context, info etl.PipelineInfo) {
	h.calls = append(h.calls, "read_start")
}

func (h *pipelineHookRecorder) OnReadEnd(ctx context.Context, info etl.PipelineInfo, records int, err error, dur time.Duration) {
	h.calls = append(h.calls, "read_end")
	h.readRecords = records
	h.readErr = err
	h.checkDuration(dur)
}

func (h *pipelineHookRecorder) OnTransformStart(ctx context.Context, info etl.PipelineInfo, step int) {
	h.calls = append(h.calls, fmt.Sprintf("transform_start_%d", step))
	h.transformSteps = append(h.transformSteps, step)
}

func (h *pipelineHookRecorder) OnTransformEnd(ctx context.Context, info etl.PipelineInfo, step int, err error, dur time.Duration) {
	h.calls = append(h.calls, fmt.Sprintf("transform_end_%d", step))
	h.transformEndErrs = append(h.transformEndErrs, err)
	h.checkDuration(dur)
}

func (h *pipelineHookRecorder) OnWriteStart(ctx context.Context, info etl.PipelineInfo, records int) {
	h.calls = append(h.calls, "write_start")
	h.writeRecords = records
}

func (h *pipelineHookRecorder) OnWriteEnd(ctx context.Context, info etl.PipelineInfo, err error, dur time.Duration) {
	h.calls = append(h.calls, "write_end")
	h.writeErr = err
	h.checkDuration(dur)
}

func (h *pipelineHookRecorder) checkDuration(dur time.Duration) {
	if dur < 0 {
		h.negativeDuration = true
	}
}

func TestPipelineHooksSuccess(t *testing.T) {
	ctx := context.Background()
	hook := &pipelineHookRecorder{}

	pipeline := etl.New("demo").
		Hook(hook).
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
		Write(func(ctx context.Context, records []etl.Record) error {
			return nil
		})

	if err := pipeline.Run(ctx); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	wantCalls := []string{
		"pipeline_start",
		"read_start",
		"read_end",
		"transform_start_0",
		"transform_end_0",
		"write_start",
		"write_end",
		"pipeline_end",
	}
	if !reflect.DeepEqual(hook.calls, wantCalls) {
		t.Fatalf("unexpected hook order: %#v", hook.calls)
	}
	if hook.info.PipelineName != "demo" {
		t.Fatalf("expected pipeline name 'demo', got %q", hook.info.PipelineName)
	}
	if hook.pipelineErr != nil {
		t.Fatalf("expected nil pipeline error, got %v", hook.pipelineErr)
	}
	if hook.readRecords != 2 {
		t.Fatalf("expected 2 records, got %d", hook.readRecords)
	}
	if hook.readErr != nil {
		t.Fatalf("expected nil read error, got %v", hook.readErr)
	}
	if len(hook.transformSteps) != 1 || hook.transformSteps[0] != 0 {
		t.Fatalf("unexpected transform steps: %#v", hook.transformSteps)
	}
	if len(hook.transformEndErrs) != 1 || hook.transformEndErrs[0] != nil {
		t.Fatalf("unexpected transform errors: %#v", hook.transformEndErrs)
	}
	if hook.writeRecords != 2 {
		t.Fatalf("expected write records 2, got %d", hook.writeRecords)
	}
	if hook.writeErr != nil {
		t.Fatalf("expected nil write error, got %v", hook.writeErr)
	}
	if hook.negativeDuration {
		t.Fatalf("expected non-negative durations")
	}
}

func TestPipelineHooksReadError(t *testing.T) {
	ctx := context.Background()
	readErr := errors.New("read broke")
	hook := &pipelineHookRecorder{}

	pipeline := etl.New("demo").
		Hook(hook).
		Read(func(ctx context.Context) ([]etl.Record, error) {
			return nil, readErr
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			return nil
		})

	err := pipeline.Run(ctx)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	wantCalls := []string{
		"pipeline_start",
		"read_start",
		"read_end",
		"pipeline_end",
	}
	if !reflect.DeepEqual(hook.calls, wantCalls) {
		t.Fatalf("unexpected hook order: %#v", hook.calls)
	}
	if hook.readErr != readErr {
		t.Fatalf("expected read error %v, got %v", readErr, hook.readErr)
	}
	if hook.pipelineErr == nil {
		t.Fatalf("expected pipeline error, got nil")
	}
	if !errors.Is(hook.pipelineErr, readErr) {
		t.Fatalf("expected pipeline error to wrap read error, got %v", hook.pipelineErr)
	}
	if hook.negativeDuration {
		t.Fatalf("expected non-negative durations")
	}
}

func TestPipelineHooksTransformError(t *testing.T) {
	ctx := context.Background()
	transformErr := errors.New("transform broke")
	hook := &pipelineHookRecorder{}

	pipeline := etl.New("demo").
		Hook(hook).
		Read(func(ctx context.Context) ([]etl.Record, error) {
			return []etl.Record{{"value": 1}}, nil
		}).
		Transform(func(ctx context.Context, record etl.Record) (etl.Record, error) {
			return nil, transformErr
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			return nil
		})

	err := pipeline.Run(ctx)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	wantCalls := []string{
		"pipeline_start",
		"read_start",
		"read_end",
		"transform_start_0",
		"transform_end_0",
		"pipeline_end",
	}
	if !reflect.DeepEqual(hook.calls, wantCalls) {
		t.Fatalf("unexpected hook order: %#v", hook.calls)
	}
	if len(hook.transformEndErrs) != 1 || hook.transformEndErrs[0] != transformErr {
		t.Fatalf("expected transform error %v, got %#v", transformErr, hook.transformEndErrs)
	}
	if hook.pipelineErr == nil {
		t.Fatalf("expected pipeline error, got nil")
	}
	if !errors.Is(hook.pipelineErr, transformErr) {
		t.Fatalf("expected pipeline error to wrap transform error, got %v", hook.pipelineErr)
	}
	if hook.negativeDuration {
		t.Fatalf("expected non-negative durations")
	}
}
