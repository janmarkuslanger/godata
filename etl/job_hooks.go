package etl

import (
	"context"
	"time"
)

type JobInfo struct {
	JobID        string
	PipelineName string
}

type JobHook interface {
	OnJobStart(ctx context.Context, info JobInfo)
	OnJobEnd(ctx context.Context, info JobInfo, err error, dur time.Duration)
}

type NoopJobHook struct{}

func (NoopJobHook) OnJobStart(context.Context, JobInfo)                     {}
func (NoopJobHook) OnJobEnd(context.Context, JobInfo, error, time.Duration) {}
