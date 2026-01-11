package etl

import (
	"context"
	"time"
)

// PipelineInfo contains metadata of the pipeline
type PipelineInfo struct {
	PipelineName string
}

// PipelineHook receives pipeline execution events
type PipelineHook interface {
	OnPipelineStart(ctx context.Context, info PipelineInfo)
	OnPipelineEnd(ctx context.Context, info PipelineInfo, err error, dur time.Duration)

	OnReadStart(ctx context.Context, info PipelineInfo)
	OnReadEnd(ctx context.Context, info PipelineInfo, records int, err error, dur time.Duration)

	OnTransformStart(ctx context.Context, info PipelineInfo, step int)
	OnTransformEnd(ctx context.Context, info PipelineInfo, step int, err error, dur time.Duration)

	OnWriteStart(ctx context.Context, info PipelineInfo, records int)
	OnWriteEnd(ctx context.Context, info PipelineInfo, err error, dur time.Duration)
}

type NoopPipelineHook struct{}

func (NoopPipelineHook) OnPipelineStart(context.Context, PipelineInfo)                           {}
func (NoopPipelineHook) OnPipelineEnd(context.Context, PipelineInfo, error, time.Duration)       {}
func (NoopPipelineHook) OnReadStart(context.Context, PipelineInfo)                               {}
func (NoopPipelineHook) OnReadEnd(context.Context, PipelineInfo, int, error, time.Duration)      {}
func (NoopPipelineHook) OnTransformStart(context.Context, PipelineInfo, int)                     {}
func (NoopPipelineHook) OnTransformEnd(context.Context, PipelineInfo, int, error, time.Duration) {}
func (NoopPipelineHook) OnWriteStart(context.Context, PipelineInfo, int)                         {}
func (NoopPipelineHook) OnWriteEnd(context.Context, PipelineInfo, error, time.Duration)          {}
