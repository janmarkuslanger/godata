package etl

import "context"

// Predicate decides whether a record should continue through the pipeline.
type Predicate func(ctx context.Context, record Record) (bool, error)

// step is an internal pipeline stage that can transform and/or drop a record.
type step interface {
	apply(ctx context.Context, record Record) (updated Record, keep bool, err error)
}

type transformStep struct {
	fn Transform
}

func (s transformStep) apply(ctx context.Context, record Record) (Record, bool, error) {
	updated, err := s.fn(ctx, record)
	if err != nil {
		return nil, false, err
	}
	return updated, true, nil
}

type filterStep struct {
	fn Predicate
}

func (s filterStep) apply(ctx context.Context, record Record) (Record, bool, error) {
	ok, err := s.fn(ctx, record)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return record, false, nil
	}
	return record, true, nil
}
