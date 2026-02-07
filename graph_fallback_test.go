package godi_test

import (
	"testing"

	"github.com/assurrussa/godi"
)

func TestBuildGraphDuplicateProvidersFallsBackToUnresolvedEntries(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		godi.NewDependency(func() string { return "a" }),
		godi.NewDependency(func() string { return "b" }),
	)

	g := godi.BuildGraph(deps)
	if len(g.Providers) != 2 {
		t.Fatalf("expected 2 provider nodes, got %d", len(g.Providers))
	}
}

func TestBuildGraphInvalidDependencyDoesNotPanic(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		godi.NewDependency(nil), // invalid constructor
	)

	g := godi.BuildGraph(deps)
	if len(g.Providers) != 1 {
		t.Fatalf("expected 1 provider node, got %d", len(g.Providers))
	}
	if g.Providers[0].Type != "" {
		t.Fatalf("expected invalid dependency to have empty type, got %q", g.Providers[0].Type)
	}
}

func TestBuildGraphRunnableIsGroupProvided(t *testing.T) {
	t.Parallel()

	deps := godi.NewSingleDependency(func() godi.Runnable { return godi.Runnable{} })
	g := godi.BuildGraph(deps)
	if len(g.Providers) != 1 {
		t.Fatalf("expected 1 provider node, got %d", len(g.Providers))
	}
	if len(g.Providers[0].Provides) != 1 {
		t.Fatalf("expected 1 provide token, got %d", len(g.Providers[0].Provides))
	}
	if g.Providers[0].Provides[0].Group != "goshared_di_runnable" {
		t.Fatalf("expected runnable group goshared_di_runnable, got %q", g.Providers[0].Provides[0].Group)
	}
}
