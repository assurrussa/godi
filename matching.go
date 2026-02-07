package godi

import (
	"fmt"
	"reflect"
)

type Matching struct {
	origin     reflect.Type
	interfaces []any
	err        error
}

// NewMatching matches concrete type to interfaces (dig.As).
func NewMatching(origin any, interfaces ...any) Matching {
	if origin == nil || reflect.TypeOf(origin).Kind() != reflect.Pointer {
		return Matching{
			err: fmt.Errorf("origin must be a pointer, got %T", origin),
		}
	}

	for i, iface := range interfaces {
		if !isPointerToInterface(iface) {
			return Matching{
				err: fmt.Errorf("interface must be pointer to interface, got %T at index %d", iface, i),
			}
		}
	}

	return Matching{
		origin:     reflect.TypeOf(origin).Elem(),
		interfaces: interfaces,
	}
}

func (m *Matching) Origin() reflect.Type {
	return m.origin
}

func (m *Matching) Error() error {
	return m.err
}
