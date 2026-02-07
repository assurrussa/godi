package godi

// Module groups dependencies and allows marking some as private to the module scope.
type Module struct {
	Name         string
	Dependencies Dependencies
}

func NewModule(name string, dependencies Dependencies) Module {
	return Module{Name: name, Dependencies: dependencies}
}
