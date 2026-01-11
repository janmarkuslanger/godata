package orchestrator_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
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
		state, ok, err := loadState(path)
		if err != nil {
			t.Fatalf("loadState failed: %v", err)
		}
		if ok {
			if record, found := state.Runs[handle.ID]; found {
				if record.Status != orchestrator.RunStatusRunning && !record.EndedAt.IsZero() {
					break
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for run status update")
		}
		time.Sleep(10 * time.Millisecond)
	}

	state, ok, err := loadState(path)
	if err != nil {
		t.Fatalf("loadState failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected state file to be present")
	}
	if _, found := state.Runs[handle.ID]; !found {
		t.Fatalf("expected run record to be present")
	}
}

func TestOrchestratorRunPersistsFailure(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.json")

	store := orchestrator.NewFileStore(path)
	r := runner.New()
	orch := orchestrator.New(store, r)

	pipeline := etl.New("demo").
		Read(func(ctx context.Context) ([]etl.Record, error) {
			return nil, errors.New("read failed")
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
		state, ok, err := loadState(path)
		if err != nil {
			t.Fatalf("loadState failed: %v", err)
		}
		if ok {
			if record, found := state.Runs[handle.ID]; found {
				if record.Status == orchestrator.RunStatusFailed && record.Error != "" {
					break
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for failed run status update")
		}
		time.Sleep(10 * time.Millisecond)
	}

	state, ok, err := loadState(path)
	if err != nil {
		t.Fatalf("loadState failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected state file to be present")
	}
	record, found := state.Runs[handle.ID]
	if !found {
		t.Fatalf("expected run record to be present")
	}
	if record.Status != orchestrator.RunStatusFailed {
		t.Fatalf("expected status %q, got %q", orchestrator.RunStatusFailed, record.Status)
	}
	if record.Error == "" {
		t.Fatalf("expected error to be set")
	}
}

func loadState(path string) (struct {
	Runs map[string]orchestrator.RunRecord `json:"runs"`
}, bool, error) {
	var state struct {
		Runs map[string]orchestrator.RunRecord `json:"runs"`
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, false, nil
		}
		return state, false, err
	}
	if len(data) == 0 {
		return state, false, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, false, err
	}
	if state.Runs == nil {
		state.Runs = make(map[string]orchestrator.RunRecord)
	}
	return state, true, nil
}
