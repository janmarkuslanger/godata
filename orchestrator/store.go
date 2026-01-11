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

// Store persists run metadata.
type Store interface {
	UpsertRun(ctx context.Context, record RunRecord) error
}
