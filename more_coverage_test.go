package godi_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"go.uber.org/dig"

	"github.com/assurrussa/godi"
)

func TestWithKeyAffectsGraphNodeIDAndKeyField(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		godi.NewDependency(func() string { return "x" }, godi.WithKey("k1")),
	)

	g := godi.BuildGraph(deps)
	if len(g.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(g.Providers))
	}
	if g.Providers[0].Key != "k1" {
		t.Fatalf("expected graph node key k1, got %q", g.Providers[0].Key)
	}
	if !strings.HasPrefix(g.Providers[0].ID, "key:k1#") {
		t.Fatalf("expected provider ID to use key, got %q", g.Providers[0].ID)
	}
}

func TestContainerGraphHelpers(t *testing.T) {
	t.Parallel()

	cnt, err := godi.NewContainer(godi.WithDependencies(
		godi.NewSingleDependency(func() string { return "x" }),
	))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	g := cnt.Graph()
	if len(g.Providers) == 0 {
		t.Fatal("expected graph to have providers")
	}

	dot := cnt.GraphDOT()
	if !strings.Contains(dot, "digraph DI") {
		t.Fatalf("expected DOT to start with digraph header, got:\n%s", dot)
	}

	dots := cnt.GraphDOTModules()
	if _, ok := dots["root"]; !ok {
		t.Fatalf("expected GraphDOTModules to contain root, got keys: %v", func() []string {
			out := make([]string, 0, len(dots))
			for k := range dots {
				out = append(out, k)
			}
			return out
		}())
	}
}

func TestWithMatchingsRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	_, err := godi.NewContainer(
		godi.WithDependencies(godi.NewSingleDependency(func() string { return "x" })),
		godi.WithMatchings("not-a-matching"),
	)
	if err == nil {
		t.Fatal("expected error for invalid matching input")
	}
}

func TestNewMatchingErrorCases(t *testing.T) {
	t.Parallel()

	m := godi.NewMatching(123, new(io.Reader))
	if m.Error() == nil {
		t.Fatal("expected error for non-pointer origin")
	}

	m2 := godi.NewMatching(new(bytes.Buffer), "bad-interface")
	if m2.Error() == nil {
		t.Fatal("expected error for invalid interface arg")
	}
}

func TestDecorateRejectsWithNameOption(t *testing.T) {
	t.Parallel()

	_, err := godi.NewContainer(godi.WithDependencies(
		godi.CollectDependencies(
			godi.NewDependency(func() string { return "base" }),
			godi.Decorate(func(s string) string { return s + "!" }, godi.WithName("n")),
		),
	))
	if err == nil {
		t.Fatal("expected error for Decorate with WithName option")
	}
}

func TestDependencyExposedTypesUnwrapsDigOut(t *testing.T) {
	t.Parallel()

	type out struct {
		dig.Out
		A string
		B int
	}

	dep := godi.NewDependency(func() out { return out{A: "x", B: 1} })
	types := dep.ExposedTypes()
	if len(types) != 2 {
		t.Fatalf("expected 2 exposed types, got %d", len(types))
	}
	got := map[string]bool{}
	for _, t0 := range types {
		got[t0.String()] = true
	}
	if !got["string"] || !got["int"] {
		t.Fatalf("expected exposed types [string int], got %v", got)
	}
}

func TestValidateCoversModuleProvidersAndDecorators(t *testing.T) {
	t.Parallel()

	module := godi.Module{
		Name: "m",
		Dependencies: godi.CollectDependencies(
			godi.NewDependency(func() string { return "mod" }, godi.Private()),
			godi.NewDependency(func(s string) int { return len(s) }),
			godi.Decorate(func(v int) int { return v + 1 }),
		),
	}

	cnt, err := godi.NewContainer(godi.WithModules(module))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	if err := cnt.Validate(); err != nil {
		t.Fatalf("expected validation to pass, got %v", err)
	}

	// Root should be able to resolve the exported int, decorated inside module scope.
	var got int
	if err := cnt.Invoke(func(v int) { got = v }); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if got != len("mod") {
		t.Fatalf("expected int %d, got %d", len("mod"), got)
	}
}
