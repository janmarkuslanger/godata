package etl

import "context"

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
