package orchestrator_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/janmarkuslanger/godata/etl"
	"github.com/janmarkuslanger/godata/orchestrator"
	"github.com/janmarkuslanger/godata/runner"
)

func TestOrchestratorRunPersistsStatus(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.json")

	store := orchestrator.NewFileStore(path)
	r := runner.New()
	orch := orchestrator.New(store, r)

	pipeline := etl.New("demo").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			return []etl.Record{{"value": 1}}, nil
		}).
		Write(func(ctx context.Context, records []etl.Record) error {
			return nil
		})

	if err := orch.Register(ctx, pipeline, nil); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	handle, err := orch.RunPipeline(ctx, "demo")
	if err != nil {
		t.Fatalf("RunPipeline failed: %v", err)
	}

	<-handle.Done()

	deadline := time.Now().Add(2 * time.Second)
	for {
		record, ok, err := store.GetRun(ctx, handle.ID)
		if err != nil {
			t.Fatalf("GetRun failed: %v", err)
		}
		if ok && record.Status != orchestrator.RunStatusRunning && !record.EndedAt.IsZero() {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for run status update")
		}
		time.Sleep(10 * time.Millisecond)
	}

	pipelineRecord, ok, err := store.GetPipeline(ctx, "demo")
	if err != nil || !ok {
		t.Fatalf("expected pipeline record to be present")
	}
	if !pipelineRecord.Enabled {
		t.Fatalf("expected pipeline to be enabled")
	}
}
