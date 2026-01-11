package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileStore persists metadata in a local JSON file.
type FileStore struct {
	path string
	mu   sync.Mutex
}

// NewFileStore creates a file-backed store at the given path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// UpsertPipeline inserts or updates a pipeline record.
func (s *FileStore) UpsertPipeline(ctx context.Context, record PipelineRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(record.Name) == "" {
		return errors.New("pipeline name is required")
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadLocked()
	if err != nil {
		return err
	}

	state.Pipelines[record.Name] = record
	return s.saveLocked(state)
}

// GetPipeline returns a pipeline record by name.
func (s *FileStore) GetPipeline(ctx context.Context, name string) (PipelineRecord, bool, error) {
	if err := ctx.Err(); err != nil {
		return PipelineRecord{}, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadLocked()
	if err != nil {
		return PipelineRecord{}, false, err
	}

	record, ok := state.Pipelines[name]
	return record, ok, nil
}

// ListPipelines returns all pipeline records.
func (s *FileStore) ListPipelines(ctx context.Context) ([]PipelineRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadLocked()
	if err != nil {
		return nil, err
	}

	pipelines := make([]PipelineRecord, 0, len(state.Pipelines))
	for _, record := range state.Pipelines {
		pipelines = append(pipelines, record)
	}
	return pipelines, nil
}

// UpsertRun inserts or updates a run record.
func (s *FileStore) UpsertRun(ctx context.Context, record RunRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(record.ID) == "" {
		return errors.New("run ID is required")
	}
	if strings.TrimSpace(record.PipelineName) == "" {
		return errors.New("pipeline name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadLocked()
	if err != nil {
		return err
	}

	state.Runs[record.ID] = record
	return s.saveLocked(state)
}

// GetRun returns a run record by ID.
func (s *FileStore) GetRun(ctx context.Context, id string) (RunRecord, bool, error) {
	if err := ctx.Err(); err != nil {
		return RunRecord{}, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadLocked()
	if err != nil {
		return RunRecord{}, false, err
	}

	record, ok := state.Runs[id]
	return record, ok, nil
}

// ListRuns returns run records filtered by pipeline name (optional).
func (s *FileStore) ListRuns(ctx context.Context, pipelineName string) ([]RunRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadLocked()
	if err != nil {
		return nil, err
	}

	runs := make([]RunRecord, 0, len(state.Runs))
	for _, record := range state.Runs {
		if pipelineName != "" && record.PipelineName != pipelineName {
			continue
		}
		runs = append(runs, record)
	}
	return runs, nil
}

type fileState struct {
	Pipelines map[string]PipelineRecord `json:"pipelines"`
	Runs      map[string]RunRecord      `json:"runs"`
}

func (s *FileStore) loadLocked() (fileState, error) {
	if s.path == "" {
		return emptyState(), errors.New("store path is required")
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyState(), nil
		}
		return fileState{}, err
	}
	if len(data) == 0 {
		return emptyState(), nil
	}

	var state fileState
	if err := json.Unmarshal(data, &state); err != nil {
		return fileState{}, err
	}
	if state.Pipelines == nil {
		state.Pipelines = make(map[string]PipelineRecord)
	}
	if state.Runs == nil {
		state.Runs = make(map[string]RunRecord)
	}
	return state, nil
}

func (s *FileStore) saveLocked(state fileState) error {
	if s.path == "" {
		return errors.New("store path is required")
	}

	if state.Pipelines == nil {
		state.Pipelines = make(map[string]PipelineRecord)
	}
	if state.Runs == nil {
		state.Runs = make(map[string]RunRecord)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-state-*.json")
	if err != nil {
		return err
	}

	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), s.path)
}

func emptyState() fileState {
	return fileState{
		Pipelines: make(map[string]PipelineRecord),
		Runs:      make(map[string]RunRecord),
	}
}
