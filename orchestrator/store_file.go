package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

type fileState struct {
	Runs map[string]RunRecord `json:"runs"`
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
	if state.Runs == nil {
		state.Runs = make(map[string]RunRecord)
	}
	return state, nil
}

func (s *FileStore) saveLocked(state fileState) error {
	if s.path == "" {
		return errors.New("store path is required")
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
		Runs: make(map[string]RunRecord),
	}
}
