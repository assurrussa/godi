package godi

type DependencyOption func(*Dependency)

// WithMatch maps dependency to interface (dig.As).
func WithMatch(a any) DependencyOption {
	return func(d *Dependency) {
		if err := d.AddMatchingInterface(a); err != nil && d.err == nil {
			d.err = err
		}
	}
}

// WithKey attaches a metadata key for diagnostics (no effect on resolution).
func WithKey(key string) DependencyOption {
	return func(d *Dependency) { d.key = &key }
}

// WithName provides dependency under a named key (dig.Name).
func WithName(name string) DependencyOption {
	return func(d *Dependency) { d.name = &name }
}

// WithGroup adds dependency to dig group.
func WithGroup(group string) DependencyOption {
	return func(d *Dependency) { d.group = &group }
}

// Private marks a dependency as private to its module scope.
func Private() DependencyOption {
	return func(d *Dependency) { d.private = true }
}
