package godi

import (
	"errors"
	"reflect"
)

type Dependency struct {
	constructor        any
	matchingInterfaces []any
	key                *string
	name               *string
	group              *string
	private            bool
	kind               dependencyKind
	err                error
}

type dependencyKind int

const (
	dependencyKindProvide dependencyKind = iota
	dependencyKindReplace
	dependencyKindDecorate
)

func NewDependency(constructor any, opts ...DependencyOption) Dependency {
	d := Dependency{constructor: constructor, kind: dependencyKindProvide}

	if err := validateConstructor(constructor); err != nil {
		d.err = err
		return d
	}

	for _, opt := range opts {
		opt(&d)
	}

	return d
}

func NewSingleDependency(constructor any, opts ...DependencyOption) Dependencies {
	return CollectDependencies(NewDependency(constructor, opts...))
}

// Replace declares an explicit replacement for a dependency slot.
func Replace(constructor any, opts ...DependencyOption) Dependency {
	d := NewDependency(constructor, opts...)
	d.kind = dependencyKindReplace
	return d
}

// Decorate declares a decorator for an existing dependency slot.
func Decorate(constructor any, opts ...DependencyOption) Dependency {
	d := NewDependency(constructor, opts...)
	d.kind = dependencyKindDecorate
	return d
}

func (d *Dependency) Type() reflect.Type {
	if d.err != nil {
		return nil
	}

	return reflect.TypeOf(d.constructor).Out(0)
}

// ExposedTypes returns the types this dependency is exposed as in the container.
// When matching interfaces are provided (dig.As), only those interfaces are exposed.
func (d *Dependency) ExposedTypes() []reflect.Type {
	if d.err != nil {
		return nil
	}

	if len(d.matchingInterfaces) > 0 {
		seen := map[reflect.Type]struct{}{}
		result := make([]reflect.Type, 0, len(d.matchingInterfaces))
		for _, iface := range d.matchingInterfaces {
			t := reflect.TypeOf(iface)
			if t == nil || t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Interface {
				continue
			}
			ifaceType := t.Elem()
			if _, ok := seen[ifaceType]; ok {
				continue
			}
			seen[ifaceType] = struct{}{}
			result = append(result, ifaceType)
		}
		return result
	}

	if slots, ok, _ := provideOutSlotsFromConstructor(d.constructor); ok {
		seen := map[reflect.Type]struct{}{}
		result := make([]reflect.Type, 0, len(slots))
		for _, slot := range slots {
			if slot.t == nil {
				continue
			}
			if _, exists := seen[slot.t]; exists {
				continue
			}
			seen[slot.t] = struct{}{}
			result = append(result, slot.t)
		}
		return result
	}

	t := d.Type()
	if t == nil {
		return nil
	}

	return []reflect.Type{t}
}

func (d *Dependency) IsRunnable() bool {
	t := d.Type()
	return t != nil && t.AssignableTo(reflect.TypeFor[Runnable]())
}

func (d *Dependency) Error() error {
	return d.err
}

func (d *Dependency) AddMatchingInterface(as ...any) error {
	for _, a := range as {
		if !isPointerToInterface(a) {
			return errors.New("matching interface must be pointer to interface")
		}
	}

	d.matchingInterfaces = append(d.matchingInterfaces, as...)
	return nil
}

type Dependencies struct {
	list []Dependency
}

func (d Dependencies) List() []Dependency {
	return d.list
}

func CollectDependencies(deps ...Dependency) Dependencies {
	return Dependencies{list: deps}
}
