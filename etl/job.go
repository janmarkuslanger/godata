package etl

import (
	"context"
	"encoding/hex"
	"errors"
	"math/rand"
	"time"
)

// Job represents a single execution
type Job struct {
	ID       string
	Pipeline *Pipeline
	Name     string

	StartedAt time.Time
	EndedAt   time.Time
	Err       error
}

// NewJob creates a new Job
func NewJob(pipeline *Pipeline) (*Job, error) {
	if pipeline == nil {
		return nil, errors.New("pipeline is nil")
	}

	return &Job{
		ID:       newJobID(),
		Pipeline: pipeline,
		Name:     pipeline.Name(),
	}, nil
}

// Duration returns the duration if the job
func (j *Job) Duration() time.Duration {
	if j.StartedAt.IsZero() {
		return 0
	}

	if j.EndedAt.IsZero() {
		return time.Since(j.StartedAt)
	}

	return j.EndedAt.Sub(j.StartedAt)
}

// Run executes the underlying pipeline
func (j *Job) Run(ctx context.Context) error {
	if j == nil {
		return errors.New("job is nil")
	}

	if j.Pipeline == nil {
		return errors.New("job pipeline is nil")
	}

	j.StartedAt = time.Now()
	err := j.Pipeline.Run(ctx)
	j.EndedAt = time.Now()
	j.Err = err
	return err
}

// newJobID create a new unique ID
func newJobID() string {
	var bytes [16]byte
	rand.Read(bytes[:])
	return hex.EncodeToString(bytes[:])
}
