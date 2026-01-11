package orchestrator

import (
	"context"
	"time"
)

// RunStatus represents the lifecycle state of a run.
type RunStatus string

const (
	RunStatusRunning   RunStatus = "running"
	RunStatusSucceeded RunStatus = "succeeded"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCanceled  RunStatus = "canceled"
)

// PipelineRecord stores persisted pipeline metadata.
type PipelineRecord struct {
	Name      string
	Enabled   bool
	UpdatedAt time.Time
}

// RunRecord stores persisted run metadata.
type RunRecord struct {
	ID           string
	PipelineName string
	JobID        string
	Status       RunStatus
	StartedAt    time.Time
	EndedAt      time.Time
	Error        string
}

// Store persists pipeline and run metadata.
type Store interface {
	UpsertPipeline(ctx context.Context, record PipelineRecord) error
	GetPipeline(ctx context.Context, name string) (PipelineRecord, bool, error)
	ListPipelines(ctx context.Context) ([]PipelineRecord, error)

	UpsertRun(ctx context.Context, record RunRecord) error
	GetRun(ctx context.Context, id string) (RunRecord, bool, error)
	ListRuns(ctx context.Context, pipelineName string) ([]RunRecord, error)
}
