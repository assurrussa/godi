package godi

type ContainerOption func(c *containerConfig)

// WithDependencies adds dependencies to the container.
func WithDependencies(d ...Dependencies) ContainerOption {
	return func(c *containerConfig) {
		for i := range d {
			c.dependencies = append(c.dependencies, d[i].List()...)
		}
	}
}

// WithMatchings applies dig.As mappings automatically for provided dependencies.
// Accepts pointers to interfaces or Matching instances.
func WithMatchings(matchings ...any) ContainerOption {
	return func(c *containerConfig) {
		c.matchings = append(c.matchings, matchings...)
	}
}

// WithModules registers module dependencies with optional privacy.
func WithModules(modules ...Module) ContainerOption {
	return func(c *containerConfig) {
		c.modules = append(c.modules, modules...)
	}
}

// WithDefaultLifecycle registers a default Lifecycle in the container.
func WithDefaultLifecycle() ContainerOption {
	return func(c *containerConfig) {
		c.defaultLifecycle = true
	}
}
