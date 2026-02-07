package godi

import (
	"errors"
	"fmt"
	"reflect"

	"go.uber.org/dig"
)

type containerConfig struct {
	dependencies     []Dependency
	matchings        []any
	modules          []Module
	defaultLifecycle bool
}

// Container wraps dig.Container with a tiny convenience layer.
type Container struct {
	dig          *dig.Container
	dependencies []Dependency
	modules      []Module
	matchings    []any
	started      bool
}

func NewContainer(opts ...ContainerOption) (*Container, error) {
	cfg := containerConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	modules := make([]Module, 0, len(cfg.modules))
	for _, module := range cfg.modules {
		if module.Name == "" {
			return nil, errors.New("module name is required")
		}
		deps, err := applyMatchingsToList(module.Dependencies.List(), cfg.matchings)
		if err != nil {
			return nil, fmt.Errorf("module %s: %w", module.Name, err)
		}
		modules = append(modules, Module{
			Name:         module.Name,
			Dependencies: CollectDependencies(deps...),
		})
	}

	if cfg.defaultLifecycle {
		cfg.dependencies = append(cfg.dependencies, NewDependency(NewLifecycle))
	}

	cnt := &Container{
		dig:          dig.New(dig.RecoverFromPanics()),
		dependencies: nil,
		modules:      modules,
		matchings:    cfg.matchings,
		started:      false,
	}

	if err := cnt.append(CollectDependencies(cfg.dependencies...)); err != nil {
		return nil, err
	}

	return cnt, nil
}

func (c *Container) Invoke(consumer any) error {
	c.started = true
	return c.dig.Invoke(consumer)
}

// Provide appends dependencies to the container.
func (c *Container) Provide(deps Dependencies) error {
	if c.started {
		return errors.New("cannot provide after container has been started")
	}
	return c.append(deps)
}

func (c *Container) Runnables() ([]Runnable, error) {
	var result []Runnable
	c.started = true
	err := c.dig.Invoke(func(r runnables) {
		result = r.Runnables
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Container) append(d Dependencies) error {
	deps, err := applyMatchingsToList(d.List(), c.matchings)
	if err != nil {
		return err
	}

	// Build on a copy so a failed provide doesn't permanently pollute container state.
	next := append([]Dependency(nil), c.dependencies...)
	next = append(next, deps...)

	orig := c.dependencies
	c.dependencies = next
	built, err := c.build(false)
	if err != nil {
		c.dependencies = orig
		return err
	}

	c.dig = built.container
	return nil
}

// Validate checks that all registered dependencies are resolvable without running constructors.
func (c *Container) Validate() error {
	built, err := c.build(true)
	if err != nil {
		return err
	}

	for _, provider := range built.rootProviders {
		if err := invokeProvider(built.container, provider.dep); err != nil {
			return err
		}
	}

	for moduleName, providers := range built.moduleProviders {
		scope := built.scopes[moduleName]
		for _, provider := range providers {
			if err := invokeProvider(scope, provider.dep); err != nil {
				return fmt.Errorf("module %s: %w", moduleName, err)
			}
		}
	}

	return nil
}

type buildResult struct {
	container       *dig.Container
	scopes          map[string]*dig.Scope
	rootProviders   []depEntry
	moduleProviders map[string][]depEntry
}

func (c *Container) build(dry bool) (*buildResult, error) {
	rootEntries := buildRootEntries(c.dependencies)

	moduleResolutions, err := buildModuleResolutions(c.modules)
	if err != nil {
		return nil, err
	}

	globalEntries := buildGlobalEntries(rootEntries, moduleResolutions)
	globalResolution, err := resolveEntries(globalEntries)
	if err != nil {
		return nil, err
	}
	if err := validateDecorators(globalResolution.decorators, globalResolution.slots); err != nil {
		return nil, err
	}

	root, scopes := buildDigContainer(c.modules, dry)
	rootProviders, err := provideRootProviders(root, globalResolution)
	if err != nil {
		return nil, err
	}

	moduleProviders, err := provideModuleProviders(scopes, moduleResolutions, globalResolution)
	if err != nil {
		return nil, err
	}

	if err := applyDecorators(root, scopes, globalResolution, moduleResolutions); err != nil {
		return nil, err
	}

	return &buildResult{
		container:       root,
		scopes:          scopes,
		rootProviders:   rootProviders,
		moduleProviders: moduleProviders,
	}, nil
}

func buildRootEntries(deps []Dependency) []depEntry {
	rootEntries := make([]depEntry, 0, len(deps))
	for i, dep := range deps {
		rootEntries = append(rootEntries, depEntry{dep: dep, idx: i})
	}
	return rootEntries
}

func buildModuleResolutions(modules []Module) (map[string]resolvedScope, error) {
	moduleResolutions := map[string]resolvedScope{}
	for _, module := range modules {
		entries := make([]depEntry, 0, len(module.Dependencies.List()))
		for i, dep := range module.Dependencies.List() {
			entries = append(entries, depEntry{dep: dep, idx: i, module: module.Name})
		}
		res, err := resolveEntries(entries)
		if err != nil {
			return nil, fmt.Errorf("module %s: %w", module.Name, err)
		}
		moduleResolutions[module.Name] = res
	}
	return moduleResolutions, nil
}

func buildGlobalEntries(rootEntries []depEntry, moduleResolutions map[string]resolvedScope) []depEntry {
	globalEntries := make([]depEntry, 0, len(rootEntries))
	globalEntries = append(globalEntries, rootEntries...)
	for moduleName, res := range moduleResolutions {
		for _, provider := range res.providers {
			if provider.dep.private {
				continue
			}
			provider.module = moduleName
			globalEntries = append(globalEntries, provider)
		}
	}
	return globalEntries
}

func buildDigContainer(modules []Module, dry bool) (*dig.Container, map[string]*dig.Scope) {
	opts := []dig.Option{dig.RecoverFromPanics()}
	if dry {
		opts = append(opts, dig.DryRun(true))
	}
	root := dig.New(opts...)

	scopes := map[string]*dig.Scope{}
	for _, module := range modules {
		scopes[module.Name] = root.Scope(module.Name)
	}
	return root, scopes
}

func provideRootProviders(root *dig.Container, resolution resolvedScope) ([]depEntry, error) {
	rootProviders := make([]depEntry, 0)
	for _, provider := range resolution.providers {
		if provider.module != "" {
			continue
		}
		if err := provideDependency(root, provider.dep, false); err != nil {
			return nil, err
		}
		rootProviders = append(rootProviders, provider)
	}
	return rootProviders, nil
}

func provideModuleProviders(
	scopes map[string]*dig.Scope,
	moduleResolutions map[string]resolvedScope,
	globalResolution resolvedScope,
) (map[string][]depEntry, error) {
	moduleProviders := map[string][]depEntry{}
	for moduleName, res := range moduleResolutions {
		availableSlots := mergeSlots(res.slots, globalResolution.slots)
		if err := validateDecorators(res.decorators, availableSlots); err != nil {
			return nil, fmt.Errorf("module %s: %w", moduleName, err)
		}

		scope := scopes[moduleName]
		for _, provider := range res.providers {
			if provider.dep.private {
				if err := provideDependency(scope, provider.dep, false); err != nil {
					return nil, err
				}
				moduleProviders[moduleName] = append(moduleProviders[moduleName], provider)
				continue
			}

			slots, err := dependencySlots(provider.dep)
			if err != nil {
				return nil, err
			}
			if !isGlobalWinner(provider, slots, globalResolution.slots) {
				continue
			}

			if err := provideDependency(scope, provider.dep, true); err != nil {
				return nil, err
			}
			moduleProviders[moduleName] = append(moduleProviders[moduleName], provider)
		}
	}

	return moduleProviders, nil
}

func applyDecorators(
	root *dig.Container,
	scopes map[string]*dig.Scope,
	globalResolution resolvedScope,
	moduleResolutions map[string]resolvedScope,
) error {
	for _, decorator := range globalResolution.decorators {
		if err := root.Decorate(decorator.dep.constructor); err != nil {
			return err
		}
	}
	for moduleName, res := range moduleResolutions {
		scope := scopes[moduleName]
		for _, decorator := range res.decorators {
			if err := scope.Decorate(decorator.dep.constructor); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyMatchingsToList(deps []Dependency, matchings []any) ([]Dependency, error) {
	result := make([]Dependency, 0, len(deps))
	for _, dependency := range deps {
		dep := dependency
		if dep.kind != dependencyKindDecorate {
			if err := applyMatchings(&dep, matchings); err != nil {
				return nil, err
			}
		}
		if err := dep.Error(); err != nil {
			return nil, fmt.Errorf("dependency error: %w", err)
		}
		result = append(result, dep)
	}
	return result, nil
}

func applyMatchings(dependency *Dependency, matchings []any) error {
	if dependency.kind == dependencyKindDecorate {
		return nil
	}

	var asInterfaces []any

	for _, matching := range matchings {
		if m, ok := matching.(Matching); ok {
			if err := m.Error(); err != nil {
				return fmt.Errorf("matching has an error: %w", err)
			}
			depType := dependency.Type()
			if depType == m.Origin() || (depType != nil && depType.Kind() == reflect.Pointer && depType.Elem() == m.Origin()) {
				asInterfaces = append(asInterfaces, m.interfaces...)
			}
			continue
		}

		if isPointerToInterface(matching) {
			t := dependency.Type()
			if t != nil && t.Implements(reflect.TypeOf(matching).Elem()) {
				asInterfaces = append(asInterfaces, matching)
			}
			continue
		}

		return fmt.Errorf("matching must be a pointer to interface or Matching, got %T", matching)
	}

	if len(asInterfaces) == 0 {
		return nil
	}

	return dependency.AddMatchingInterface(asInterfaces...)
}

func provideDependency(scope interface {
	Provide(constructor any, opts ...dig.ProvideOption) error
}, dep Dependency, export bool,
) error {
	if dep.kind == dependencyKindDecorate {
		return errors.New("decorate dependencies cannot be provided")
	}

	if dep.kind != dependencyKindProvide && dependencyGroup(dep) != "" {
		return errors.New("replace is not supported for group dependencies")
	}

	if dep.name != nil && dependencyGroup(dep) != "" {
		return errors.New("invalid dependency options: WithName cannot be used with WithGroup or Runnable")
	}
	if dep.IsRunnable() && dep.group != nil {
		return errors.New("invalid dependency options: Runnable cannot be used with WithGroup")
	}

	options := make([]dig.ProvideOption, 0)
	for _, as := range dep.matchingInterfaces {
		options = append(options, dig.As(as))
	}
	if dep.group != nil {
		options = append(options, dig.Group(*dep.group))
	}
	if dep.IsRunnable() {
		options = append(options, dig.Group(runnableGroup))
	}
	if dep.name != nil {
		options = append(options, dig.Name(*dep.name))
	}
	if export {
		options = append(options, dig.Export(true))
	}

	return scope.Provide(dep.constructor, options...)
}

func invokeProvider(scope interface {
	Invoke(function any, opts ...dig.InvokeOption) error
}, dep Dependency,
) error {
	slots, err := dependencySlots(dep)
	if err != nil {
		return err
	}
	if len(slots) == 0 {
		return nil
	}
	for _, slot := range slots {
		fn, err := buildValidationInvokeForSlot(slot)
		if err != nil {
			return err
		}
		if err := scope.Invoke(fn); err != nil {
			return err
		}
	}
	return nil
}

func validateDecorators(decorators []depEntry, available map[slotKey]depEntry) error {
	for _, decorator := range decorators {
		slots, err := decoratorSlots(decorator.dep)
		if err != nil {
			return err
		}
		for _, slot := range slots {
			if _, ok := available[slot]; !ok {
				return fmt.Errorf("cannot decorate slot %s: no provider", slotLabel(slot))
			}
		}
	}
	return nil
}

func mergeSlots(a, b map[slotKey]depEntry) map[slotKey]depEntry {
	out := map[slotKey]depEntry{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func isGlobalWinner(entry depEntry, slots []slotKey, globalSlots map[slotKey]depEntry) bool {
	for _, slot := range slots {
		if slot.group != "" {
			continue
		}
		winner, ok := globalSlots[slot]
		if !ok || !sameEntry(winner, entry) {
			return false
		}
	}
	return true
}

//nolint:unparam // it's valid
func buildValidationInvokeForSlot(slot slotKey) (any, error) {
	if slot.t == nil {
		return func() {}, nil
	}

	if slot.group != "" {
		return buildGroupInvoke(slot.t, slot.group), nil
	}

	if slot.name != "" {
		return buildNamedInvoke(slot.t, slot.name), nil
	}

	fnType := reflect.FuncOf([]reflect.Type{slot.t}, nil, false)
	fn := reflect.MakeFunc(fnType, func([]reflect.Value) []reflect.Value { return nil })
	return fn.Interface(), nil
}

func buildGroupInvoke(t reflect.Type, group string) any {
	inType := reflect.StructOf([]reflect.StructField{
		{
			Name:      "In",
			Type:      reflect.TypeOf(dig.In{}),
			Anonymous: true,
		},
		{
			Name: "Items",
			Type: reflect.SliceOf(t),
			Tag:  reflect.StructTag(fmt.Sprintf(`group:"%s"`, group)),
		},
	})

	fnType := reflect.FuncOf([]reflect.Type{inType}, nil, false)
	fn := reflect.MakeFunc(fnType, func([]reflect.Value) []reflect.Value { return nil })
	return fn.Interface()
}

func buildNamedInvoke(t reflect.Type, name string) any {
	inType := reflect.StructOf([]reflect.StructField{
		{
			Name:      "In",
			Type:      reflect.TypeOf(dig.In{}),
			Anonymous: true,
		},
		{
			Name: "Item",
			Type: t,
			Tag:  reflect.StructTag(fmt.Sprintf(`name:"%s"`, name)),
		},
	})

	fnType := reflect.FuncOf([]reflect.Type{inType}, nil, false)
	fn := reflect.MakeFunc(fnType, func([]reflect.Value) []reflect.Value { return nil })
	return fn.Interface()
}
