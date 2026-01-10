package etl

import "context"

type Record map[string]any

type Reader func(ctx context.Context) ([]Record, error)

type Transform func(ctx context.Context, record Record) (Record, error)

type Writer func(ctx context.Context, records []Record) error

type Pipeline struct {
	name       string
	reader     Reader
	transforms []Transform
	writer     Writer
}

func New(name string) *Pipeline {
	return &Pipeline{name: name}
}

func (p *Pipeline) Name() string {
	return p.name
}

func (p *Pipeline) Read(reader Reader) *Pipeline {
	p.reader = reader
	return p
}

func (p *Pipeline) Transform(transform Transform) *Pipeline {
	p.transforms = append(p.transforms, transform)
	return p
}

func (p *Pipeline) Write(writer Writer) *Pipeline {
	p.writer = writer
	return p
}
