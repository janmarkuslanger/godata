package orchestrator_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/janmarkuslanger/godata/orchestrator"
)

func TestFileStorePersistsRecords(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.json")

	store := orchestrator.NewFileStore(path)

	pipeline := orchestrator.PipelineRecord{
		Name:    "demo",
		Enabled: true,
	}
	if err := store.UpsertPipeline(ctx, pipeline); err != nil {
		t.Fatalf("UpsertPipeline failed: %v", err)
	}

	run := orchestrator.RunRecord{
		ID:           "run-1",
		PipelineName: "demo",
		Status:       orchestrator.RunStatusRunning,
	}
	if err := store.UpsertRun(ctx, run); err != nil {
		t.Fatalf("UpsertRun failed: %v", err)
	}

	if _, ok, err := store.GetPipeline(ctx, "demo"); err != nil || !ok {
		t.Fatalf("expected pipeline to be present")
	}
	if _, ok, err := store.GetRun(ctx, "run-1"); err != nil || !ok {
		t.Fatalf("expected run to be present")
	}

	storeReloaded := orchestrator.NewFileStore(path)
	if _, ok, err := storeReloaded.GetPipeline(ctx, "demo"); err != nil || !ok {
		t.Fatalf("expected pipeline to be present after reload")
	}
	if _, ok, err := storeReloaded.GetRun(ctx, "run-1"); err != nil || !ok {
		t.Fatalf("expected run to be present after reload")
	}
}
