package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/janmarkuslanger/godata/etl"
)

var (
	ErrRunnerNil   = errors.New("runner is nil")
	ErrJobNil      = errors.New("job is nil")
	ErrRunNotFound = errors.New("run not found")
)

// Status describes the outcome of a run.
type Status string

const (
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

// Result captures the outcome of a run.
type Result struct {
	ID        string
	JobID     string
	JobName   string
	Status    Status
	StartedAt time.Time
	EndedAt   time.Time
	Err       error
}

// Handle represents a running or completed job.
type Handle struct {
	ID        string
	JobID     string
	JobName   string
	StartedAt time.Time

	done   <-chan struct{}
	runner *Runner
}

// Done returns a channel that is closed when the run completes.
func (h Handle) Done() <-chan struct{} {
	return h.done
}

// Result returns the run result if it is available.
func (h Handle) Result() (Result, bool) {
	if h.runner == nil {
		return Result{}, false
	}
	return h.runner.Result(h.ID)
}

type runState struct {
	cancel    context.CancelFunc
	done      chan struct{}
	jobID     string
	jobName   string
	startedAt time.Time
}

// Runner executes jobs and tracks their status.
type Runner struct {
	mu      sync.Mutex
	runs    map[string]*runState
	results map[string]Result
}

// New creates a Runner instance.
func New() *Runner {
	return &Runner{
		runs:    make(map[string]*runState),
		results: make(map[string]Result),
	}
}

// Start runs a job asynchronously and returns a handle.
func (r *Runner) Start(ctx context.Context, job *etl.Job) (Handle, error) {
	if r == nil {
		return Handle{}, ErrRunnerNil
	}
	if job == nil {
		return Handle{}, ErrJobNil
	}

	r.mu.Lock()
	if r.runs == nil {
		r.runs = make(map[string]*runState)
	}
	if r.results == nil {
		r.results = make(map[string]Result)
	}
	id := newID()
	startedAt := time.Now().UTC()
	state := &runState{
		cancel:    nil,
		done:      make(chan struct{}),
		jobID:     job.ID,
		jobName:   job.Name,
		startedAt: startedAt,
	}
	r.runs[id] = state
	r.mu.Unlock()

	runCtx, cancel := context.WithCancel(ctx)
	state.cancel = cancel

	handle := Handle{
		ID:        id,
		JobID:     job.ID,
		JobName:   job.Name,
		StartedAt: startedAt,
		done:      state.done,
		runner:    r,
	}

	go r.run(id, runCtx, job, state)

	return handle, nil
}

// Stop requests cancellation of a running job.
func (r *Runner) Stop(id string) error {
	if r == nil {
		return ErrRunnerNil
	}

	r.mu.Lock()
	state, ok := r.runs[id]
	r.mu.Unlock()
	if !ok {
		return ErrRunNotFound
	}

	if state.cancel != nil {
		state.cancel()
	}
	return nil
}

// Result returns the latest known result for a run.
func (r *Runner) Result(id string) (Result, bool) {
	if r == nil {
		return Result{}, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if state, ok := r.runs[id]; ok {
		return Result{
			ID:        id,
			JobID:     state.jobID,
			JobName:   state.jobName,
			Status:    StatusRunning,
			StartedAt: state.startedAt,
		}, true
	}
	result, ok := r.results[id]
	return result, ok
}

// Active returns all running jobs.
func (r *Runner) Active() []Result {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	active := make([]Result, 0, len(r.runs))
	for id, state := range r.runs {
		active = append(active, Result{
			ID:        id,
			JobID:     state.jobID,
			JobName:   state.jobName,
			Status:    StatusRunning,
			StartedAt: state.startedAt,
		})
	}
	return active
}

func (r *Runner) run(id string, ctx context.Context, job *etl.Job, state *runState) {
	err := job.Run(ctx)
	endedAt := time.Now().UTC()
	status := statusFromErr(err)

	result := Result{
		ID:        id,
		JobID:     job.ID,
		JobName:   job.Name,
		Status:    status,
		StartedAt: state.startedAt,
		EndedAt:   endedAt,
		Err:       err,
	}

	r.mu.Lock()
	delete(r.runs, id)
	if r.results == nil {
		r.results = make(map[string]Result)
	}
	r.results[id] = result
	done := state.done
	r.mu.Unlock()

	close(done)
}

func statusFromErr(err error) Status {
	if err == nil {
		return StatusSucceeded
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return StatusCanceled
	}
	return StatusFailed
}

func newID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buf[:])
}
