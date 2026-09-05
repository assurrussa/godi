package godi

import "go.uber.org/dig"

// Optional allows resolving dependency if it exists.
type Optional[T any] struct {
	dig.In
	Optional *T `optional:"true"`
}

// Get returns a copy of the value. Use GetPtr for objects containing locks or
// when the identity of the provided instance must be preserved.
func (o *Optional[T]) Get() (T, bool) {
	if o.Optional == nil {
		var zero T
		return zero, false
	}

	return *o.Optional, true
}

// GetPtr returns the original pointer without copying the provided value.
func (o *Optional[T]) GetPtr() (*T, bool) {
	return o.Optional, o.Optional != nil
}
