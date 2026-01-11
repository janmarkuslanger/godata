package orchestrator_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/janmarkuslanger/godata/orchestrator"
)

func TestFileStorePersistsRecords(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.json")

	store := orchestrator.NewFileStore(path)

	run := orchestrator.RunRecord{
		ID:           "run-1",
		PipelineName: "demo",
		Status:       orchestrator.RunStatusRunning,
	}
	if err := store.UpsertRun(ctx, run); err != nil {
		t.Fatalf("UpsertRun failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var state struct {
		Runs map[string]orchestrator.RunRecord `json:"runs"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if _, ok := state.Runs["run-1"]; !ok {
		t.Fatalf("expected run to be persisted")
	}
}
