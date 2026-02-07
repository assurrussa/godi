package godi_test

import (
	"context"
	"testing"

	"github.com/assurrussa/godi"
)

func TestRunnablesCollectsGroup(t *testing.T) {
	t.Parallel()

	cnt, err := godi.NewContainer(godi.WithDependencies(
		godi.CollectDependencies(
			godi.NewDependency(func() godi.Runnable {
				return godi.Runnable{
					OnStart: func(context.Context) error { return nil },
					OnStop:  func(context.Context) error { return nil },
				}
			}),
			godi.NewDependency(func() godi.Runnable {
				return godi.Runnable{
					OnStart: func(context.Context) error { return nil },
					OnStop:  func(context.Context) error { return nil },
				}
			}),
		),
	))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	r, err := cnt.Runnables()
	if err != nil {
		t.Fatalf("Runnables error: %v", err)
	}
	if len(r) != 2 {
		t.Fatalf("expected 2 runnables, got %d", len(r))
	}

	// Runnables() starts the container, further Provide must fail.
	if err := cnt.Provide(godi.NewSingleDependency(func() int { return 1 })); err == nil {
		t.Fatal("expected Provide to fail after Runnables()")
	}
}

func TestWithDefaultLifecycleProvidesLifecycle(t *testing.T) {
	t.Parallel()

	cnt, err := godi.NewContainer(
		godi.WithDefaultLifecycle(),
	)
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	var got *godi.Lifecycle
	if err := cnt.Invoke(func(l *godi.Lifecycle) { got = l }); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if got == nil {
		t.Fatal("expected default lifecycle to be provided")
	}
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	deps := godi.NewSingleDependency(func() string { return "x" })
	m := godi.NewModule("m", deps)
	if m.Name != "m" {
		t.Fatalf("expected module name 'm', got %q", m.Name)
	}
	if len(m.Dependencies.List()) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(m.Dependencies.List()))
	}
}
