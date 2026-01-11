package etl_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/janmarkuslanger/godata/etl"
)

type jobHookRecorder struct {
	starts           int
	ends             int
	startInfo        etl.JobInfo
	endInfo          etl.JobInfo
	endErr           error
	endDuration      time.Duration
	negativeDuration bool
}

func (h *jobHookRecorder) OnJobStart(ctx context.Context, info etl.JobInfo) {
	h.starts++
	h.startInfo = info
}

func (h *jobHookRecorder) OnJobEnd(ctx context.Context, info etl.JobInfo, err error, dur time.Duration) {
	h.ends++
	h.endInfo = info
	h.endErr = err
	h.endDuration = dur
	if dur < 0 {
		h.negativeDuration = true
	}
}

func TestNewJobValidation(t *testing.T) {
	job, err := etl.NewJob(nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if job != nil {
		t.Fatalf("expected nil job, got %#v", job)
	}
}

func TestJobRunNilReceiver(t *testing.T) {
	ctx := context.Background()
	var job *etl.Job

	err := job.Run(ctx)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "job is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJobRunNilPipeline(t *testing.T) {
	ctx := context.Background()
	job := &etl.Job{}

	err := job.Run(ctx)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "job pipeline is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJobRunSuccess(t *testing.T) {
	ctx := context.Background()
	hook := &jobHookRecorder{}

	pipeline := etl.New("demo").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			return []etl.Record{{"value": 1}}, nil
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			return nil
		})

	job, err := etl.NewJob(pipeline)
	if err != nil {
		t.Fatalf("NewJob failed: %v", err)
	}
	job.Hook(hook)

	if err := job.Run(ctx); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if job.ID == "" {
		t.Fatalf("expected job ID")
	}
	if job.Name != "demo" {
		t.Fatalf("expected job name 'demo', got %q", job.Name)
	}
	if job.Err != nil {
		t.Fatalf("expected nil job error, got %v", job.Err)
	}
	if job.StartedAt.IsZero() || job.EndedAt.IsZero() {
		t.Fatalf("expected start/end timestamps to be set")
	}
	if job.EndedAt.Before(job.StartedAt) {
		t.Fatalf("expected end >= start")
	}

	if hook.starts != 1 || hook.ends != 1 {
		t.Fatalf("expected start/end hooks to run once, got %d/%d", hook.starts, hook.ends)
	}
	if hook.startInfo.JobID != job.ID || hook.endInfo.JobID != job.ID {
		t.Fatalf("expected hook job ID to match")
	}
	if hook.startInfo.PipelineName != "demo" || hook.endInfo.PipelineName != "demo" {
		t.Fatalf("expected hook pipeline name 'demo'")
	}
	if hook.endErr != nil {
		t.Fatalf("expected nil hook error, got %v", hook.endErr)
	}
	if hook.negativeDuration {
		t.Fatalf("expected non-negative duration")
	}
}

func TestJobRunError(t *testing.T) {
	ctx := context.Background()
	readErr := errors.New("read broke")
	hook := &jobHookRecorder{}

	pipeline := etl.New("demo").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			return nil, readErr
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			return nil
		})

	job, err := etl.NewJob(pipeline)
	if err != nil {
		t.Fatalf("NewJob failed: %v", err)
	}
	job.Hook(hook)

	err = job.Run(ctx)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if job.Err == nil {
		t.Fatalf("expected job.Err to be set")
	}
	if !errors.Is(job.Err, readErr) {
		t.Fatalf("expected job.Err to wrap read error, got %v", job.Err)
	}
	if hook.endErr == nil {
		t.Fatalf("expected hook error, got nil")
	}
	if !errors.Is(hook.endErr, readErr) {
		t.Fatalf("expected hook error to wrap read error, got %v", hook.endErr)
	}
}

func TestJobDuration(t *testing.T) {
	if dur := (&etl.Job{}).Duration(); dur != 0 {
		t.Fatalf("expected zero duration, got %v", dur)
	}

	start := time.Date(2024, time.January, 1, 10, 0, 0, 0, time.UTC)
	job := &etl.Job{StartedAt: start, EndedAt: start.Add(2 * time.Second)}
	if dur := job.Duration(); dur != 2*time.Second {
		t.Fatalf("expected 2s duration, got %v", dur)
	}
}
