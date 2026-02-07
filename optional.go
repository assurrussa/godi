package godi

import "go.uber.org/dig"

// Optional allows resolving dependency if it exists.
type Optional[T any] struct {
	dig.In
	Optional *T `optional:"true"`
}

func (o *Optional[T]) Get() (T, bool) {
	if o.Optional == nil {
		var zero T
		return zero, false
	}

	return *o.Optional, true
}
