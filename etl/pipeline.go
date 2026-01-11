package etl

import (
	"context"
	"errors"
	"fmt"
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
	name       string
	reader     Reader
	transforms []Transform
	writer     Writer
}

// New create a new pipeline
func New(name string) *Pipeline {
	return &Pipeline{name: name}
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
	p.transforms = append(p.transforms, transform)
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

// Run executes the pipeline
func (p *Pipeline) Run(ctx context.Context) error {
	if err := p.validate(); err != nil {
		return err
	}

	records, err := p.reader(ctx)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}

	// Transforms sequentially
	for i := range records {
		record := records[i]

		for step, transform := range p.transforms {
			updatedRecord, err := transform(ctx, record)
			if err != nil {
				return fmt.Errorf("transform[%d] failed: %w", step, err)
			}

			record = updatedRecord
		}

		records[i] = record
	}

	if err := p.writer(ctx, records); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	return nil
}
