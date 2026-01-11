package runner_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/janmarkuslanger/godata/etl"
	"github.com/janmarkuslanger/godata/runner"
)

func TestRunnerStartAndResult(t *testing.T) {
	ctx := context.Background()

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

	r := runner.New()
	handle, err := r.Start(ctx, job)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	<-handle.Done()
	if handle.JobID != job.ID {
		t.Fatalf("expected job ID %q, got %q", job.ID, handle.JobID)
	}
	if handle.JobName != job.Name {
		t.Fatalf("expected job name %q, got %q", job.Name, handle.JobName)
	}

	result, ok := r.Result(handle.ID)
	if !ok {
		t.Fatalf("expected result to be available")
	}
	if result.Status != runner.StatusSucceeded {
		t.Fatalf("expected status %q, got %q", runner.StatusSucceeded, result.Status)
	}
	if result.Err != nil {
		t.Fatalf("expected no error, got %v", result.Err)
	}
}

func TestRunnerAllowsOverlapping(t *testing.T) {
	ctx := context.Background()
	block := make(chan struct{})

	pipeline := etl.New("demo").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-block:
				return []etl.Record{{"value": 1}}, nil
			}
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			return nil
		})

	jobA, err := etl.NewJob(pipeline)
	if err != nil {
		t.Fatalf("NewJob failed: %v", err)
	}
	jobB, err := etl.NewJob(pipeline)
	if err != nil {
		t.Fatalf("NewJob failed: %v", err)
	}

	r := runner.New()
	handleA, err := r.Start(ctx, jobA)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	handleB, err := r.Start(ctx, jobB)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	close(block)

	select {
	case <-handleA.Done():
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for handleA")
	}
	select {
	case <-handleB.Done():
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for handleB")
	}
}

func TestRunnerResultFailure(t *testing.T) {
	ctx := context.Background()

	pipeline := etl.New("demo").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			return nil, errors.New("read failed")
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			return nil
		})

	job, err := etl.NewJob(pipeline)
	if err != nil {
		t.Fatalf("NewJob failed: %v", err)
	}

	r := runner.New()
	handle, err := r.Start(ctx, job)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	<-handle.Done()

	result, ok := r.Result(handle.ID)
	if !ok {
		t.Fatalf("expected result to be available")
	}
	if result.Status != runner.StatusFailed {
		t.Fatalf("expected status %q, got %q", runner.StatusFailed, result.Status)
	}
	if result.Err == nil {
		t.Fatalf("expected error to be set")
	}
}
