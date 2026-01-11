package orchestrator

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/janmarkuslanger/godata/etl"
	"github.com/janmarkuslanger/godata/runner"
	"github.com/janmarkuslanger/godata/scheduler"
)

var (
	ErrAlreadyRegistered = errors.New("pipeline already registered")
	ErrUnknownPipeline   = errors.New("pipeline not registered")
	ErrSchedulerRunning  = errors.New("scheduler already running")
)

// Orchestrator coordinates pipelines, schedules, and persistence.
type Orchestrator struct {
	store  Store
	runner *runner.Runner

	mu        sync.Mutex
	pipelines map[string]*etl.Pipeline
	schedules map[string]scheduler.Schedule
	schedCtx  context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// New creates a new Orchestrator.
func New(store Store, runnerInstance *runner.Runner) *Orchestrator {
	if runnerInstance == nil {
		runnerInstance = runner.New()
	}
	return &Orchestrator{
		store:     store,
		runner:    runnerInstance,
		pipelines: make(map[string]*etl.Pipeline),
		schedules: make(map[string]scheduler.Schedule),
	}
}

// Register adds a pipeline and optional schedule.
func (o *Orchestrator) Register(ctx context.Context, pipeline *etl.Pipeline, schedule scheduler.Schedule) error {
	if pipeline == nil {
		return errors.New("pipeline is nil")
	}
	name := pipeline.Name()
	if name == "" {
		return errors.New("pipeline name is required")
	}

	o.mu.Lock()
	if _, exists := o.pipelines[name]; exists {
		o.mu.Unlock()
		return ErrAlreadyRegistered
	}
	o.pipelines[name] = pipeline
	o.schedules[name] = schedule
	schedulerRunning := o.cancel != nil
	schedCtx := o.schedCtx
	o.mu.Unlock()

	if schedulerRunning && schedule != nil {
		o.startSchedule(schedCtx, name, schedule)
	}

	return nil
}

// RunPipeline starts a pipeline immediately and persists the run status.
func (o *Orchestrator) RunPipeline(ctx context.Context, name string) (runner.Handle, error) {
	pipeline, err := o.pipeline(name)
	if err != nil {
		return runner.Handle{}, err
	}

	job, err := etl.NewJob(pipeline)
	if err != nil {
		return runner.Handle{}, err
	}

	handle, err := o.runner.Start(ctx, job)
	if err != nil {
		return runner.Handle{}, err
	}

	if o.store != nil {
		record := RunRecord{
			ID:           handle.ID,
			PipelineName: name,
			JobID:        handle.JobID,
			Status:       RunStatusRunning,
			StartedAt:    handle.StartedAt,
		}
		if err := o.store.UpsertRun(context.Background(), record); err != nil {
			return runner.Handle{}, err
		}
	}

	go o.persistCompletion(name, handle)

	return handle, nil
}

// StartScheduler begins scheduling all registered pipelines.
func (o *Orchestrator) StartScheduler(ctx context.Context) error {
	o.mu.Lock()
	if o.cancel != nil {
		o.mu.Unlock()
		return ErrSchedulerRunning
	}

	schedCtx, cancel := context.WithCancel(ctx)
	o.schedCtx = schedCtx
	o.cancel = cancel

	schedules := make(map[string]scheduler.Schedule, len(o.schedules))
	for name, schedule := range o.schedules {
		schedules[name] = schedule
	}
	o.mu.Unlock()

	for name, schedule := range schedules {
		if schedule == nil {
			continue
		}
		o.startSchedule(schedCtx, name, schedule)
	}

	return nil
}

func (o *Orchestrator) pipeline(name string) (*etl.Pipeline, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	pipeline, ok := o.pipelines[name]
	if !ok {
		return nil, ErrUnknownPipeline
	}
	return pipeline, nil
}

func (o *Orchestrator) startSchedule(ctx context.Context, name string, schedule scheduler.Schedule) {
	o.wg.Add(1)
	go o.runSchedule(ctx, name, schedule)
}

func (o *Orchestrator) runSchedule(ctx context.Context, name string, schedule scheduler.Schedule) {
	defer o.wg.Done()

	last := time.Now().UTC()
	for {
		next := schedule.Next(last)
		if next.IsZero() {
			return
		}

		delay := time.Until(next)
		if delay < 0 {
			delay = 0
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		if _, err := o.RunPipeline(context.Background(), name); err != nil {
			_ = err
		}

		last = next
	}
}

func (o *Orchestrator) persistCompletion(name string, handle runner.Handle) {
	<-handle.Done()

	if o.store == nil {
		return
	}

	record := RunRecord{
		ID:           handle.ID,
		PipelineName: name,
		JobID:        handle.JobID,
	}

	if result, ok := o.runner.Result(handle.ID); ok {
		record.Status = mapRunStatus(result.Status)
		record.StartedAt = result.StartedAt
		record.EndedAt = result.EndedAt
		if result.Err != nil {
			record.Error = result.Err.Error()
		}
	} else {
		record.Status = RunStatusFailed
		record.StartedAt = handle.StartedAt
		record.EndedAt = time.Now().UTC()
		record.Error = "run result not found"
	}

	ctx := context.Background()
	if err := o.store.UpsertRun(ctx, record); err != nil {
		_ = err
	}
}

func mapRunStatus(status runner.Status) RunStatus {
	switch status {
	case runner.StatusSucceeded:
		return RunStatusSucceeded
	case runner.StatusFailed:
		return RunStatusFailed
	case runner.StatusCanceled:
		return RunStatusCanceled
	case runner.StatusRunning:
		return RunStatusRunning
	default:
		return RunStatusFailed
	}
}
