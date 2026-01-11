package etl

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Record is the data structure folowing through the pipeline
type Record map[string]any

// Reader loads records via batch
type Reader func(ctx context.Context) ([]Record, error)

// Transform processes a single record and returns the updated record
type Transform func(ctx context.Context, record Record) (Record, error)

// Writer writes the resulting records
type Writer func(ctx context.Context, records []Record) error

// Pipeline defines a data pipeline: Read -> Transform* -> Write
type Pipeline struct {
	name   string
	reader Reader
	steps  []step
	writer Writer

	hook PipelineHook
}

// New create a new pipeline
func New(name string) *Pipeline {
	return &Pipeline{name: name, hook: NoopPipelineHook{}}
}

// Name returns the name of the pipeline
func (p *Pipeline) Name() string {
	return p.name
}

// Read sets the reader
func (p *Pipeline) Read(reader Reader) *Pipeline {
	p.reader = reader
	return p
}

// Transform appends a tranformer
func (p *Pipeline) Transform(transform Transform) *Pipeline {
	p.steps = append(p.steps, transformStep{fn: transform})
	return p
}

// Filter appends a filter step.
func (p *Pipeline) Filter(predicate Predicate) *Pipeline {
	p.steps = append(p.steps, filterStep{fn: predicate})
	return p
}

// Write sets a new writer
func (p *Pipeline) Write(writer Writer) *Pipeline {
	p.writer = writer
	return p
}

// validate checks for pipxeline correctness
func (p *Pipeline) validate() error {
	if p == nil {
		return errors.New("pipeline is nil")
	}

	if p.name == "" {
		return errors.New("pipeline name must not be empty")
	}

	if p.reader == nil {
		return errors.New("reader must be set via Read")
	}

	if p.writer == nil {
		return errors.New("writer must be set via Write")
	}

	return nil
}

// Hooks sets the hook
func (p *Pipeline) Hook(hook PipelineHook) *Pipeline {
	if hook == nil {
		p.hook = NoopPipelineHook{}
		return p
	}
	p.hook = hook
	return p
}

// Run executes the pipeline
func (p *Pipeline) Run(ctx context.Context) error {
	if err := p.validate(); err != nil {
		return err
	}

	info := PipelineInfo{PipelineName: p.name}
	pipelineStart := time.Now()
	p.hook.OnPipelineStart(ctx, info)

	var runErr error
	defer func() {
		p.hook.OnPipelineEnd(ctx, info, runErr, time.Since(pipelineStart))
	}()

	readStart := time.Now()
	p.hook.OnReadStart(ctx, info)

	records, err := p.reader(ctx)
	p.hook.OnReadEnd(ctx, info, len(records), err, time.Since(readStart))

	if err != nil {
		runErr = fmt.Errorf("read failed: %w", err)
		return runErr
	}

	// Apply steps sequentially; filter steps may drop records.
	out := make([]Record, 0, len(records))

	for i := range records {
		record := records[i]
		keep := true

		for stepIdx, s := range p.steps {
			stepStart := time.Now()

			p.hook.OnStepStart(ctx, info, stepIdx)

			var err error
			record, keep, err = s.apply(ctx, record)

			p.hook.OnStepEnd(ctx, info, stepIdx, err, time.Since(stepStart))
			if err != nil {
				runErr = fmt.Errorf("step[%d] failed: %w", stepIdx, err)
				return runErr
			}

			if !keep {
				break
			}
		}

		if keep {
			out = append(out, record)
		}
	}

	writeStart := time.Now()
	p.hook.OnWriteStart(ctx, info, len(out))

	err = p.writer(ctx, out)

	p.hook.OnWriteEnd(ctx, info, err, time.Since(writeStart))
	if err != nil {
		runErr = fmt.Errorf("write failed: %w", err)
		return runErr
	}

	runErr = nil
	return nil
}
