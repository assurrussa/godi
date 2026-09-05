package godi_test

import (
	"reflect"
	"strings"
	"testing"

	"go.uber.org/dig"

	"github.com/assurrussa/godi"
)

const (
	reviewRootScope   = "root"
	reviewReplacement = "replacement"
)

type reviewOutputs struct {
	dig.Out
	Text  string
	Count int
}

func TestPartialReplaceRejectedAcrossScopes(t *testing.T) {
	t.Parallel()
	base := godi.NewDependency(func() reviewOutputs { return reviewOutputs{Text: testBase, Count: 7} })
	replacement := godi.Replace(func() string { return reviewReplacement })
	for _, placement := range []string{reviewRootScope, "module", "private", "exported", "cross-module"} {
		t.Run(placement, func(t *testing.T) {
			t.Parallel()
			var opts []godi.ContainerOption
			switch placement {
			case reviewRootScope:
				opts = append(opts, godi.WithDependencies(godi.CollectDependencies(base, replacement)))
			case "module":
				opts = append(opts, godi.WithModules(godi.NewModule("a", godi.CollectDependencies(base, replacement))))
			case "private":
				privateBase := godi.NewDependency(func() reviewOutputs { return reviewOutputs{} }, godi.Private())
				privateReplace := godi.Replace(func() string { return "x" }, godi.Private())
				opts = append(opts, godi.WithModules(godi.NewModule("a", godi.CollectDependencies(privateBase, privateReplace))))
			case "exported":
				opts = append(opts, godi.WithModules(godi.NewModule("a", godi.CollectDependencies(base))),
					godi.WithDependencies(godi.CollectDependencies(replacement)))
			case "cross-module":
				opts = append(opts, godi.WithModules(godi.NewModule("a", godi.CollectDependencies(base)),
					godi.NewModule("b", godi.CollectDependencies(replacement))))
			}
			if _, err := godi.NewContainer(opts...); err == nil || !strings.Contains(err.Error(), "partial replace") {
				t.Fatalf("expected partial replace error, got %v", err)
			}
		})
	}
}

func TestCompleteMultiOutputReplacement(t *testing.T) {
	t.Parallel()
	for _, module := range []bool{false, true} {
		for _, reverse := range []bool{false, true} {
			calls := 0
			base := godi.NewDependency(func() reviewOutputs { calls++; return reviewOutputs{} })
			replacements := []godi.Dependency{
				godi.Replace(func() string { return reviewReplacement }),
				godi.Replace(func() int { return 9 }),
			}
			var opts []godi.ContainerOption
			switch {
			case module:
				opts = append(opts, godi.WithModules(godi.NewModule("a", godi.CollectDependencies(base))))
			case reverse:
				replacements = append(replacements, base)
			default:
				replacements = append([]godi.Dependency{base}, replacements...)
			}
			opts = append(opts, godi.WithDependencies(godi.CollectDependencies(replacements...)))
			cnt, err := godi.NewContainer(opts...)
			if err != nil {
				t.Fatal(err)
			}
			if err := cnt.Validate(); err != nil {
				t.Fatal(err)
			}
			if err := cnt.Invoke(func(s string, n int) {
				if s != reviewReplacement || n != 9 {
					t.Fatalf("wrong replacement: %q %d", s, n)
				}
			}); err != nil {
				t.Fatal(err)
			}
			if calls != 0 {
				t.Fatal("replaced constructor executed")
			}
		}
	}
}

func TestFailedPartialProvidePreservesContainer(t *testing.T) {
	t.Parallel()
	cnt, err := godi.NewContainer(godi.WithDependencies(godi.NewSingleDependency(func() reviewOutputs {
		return reviewOutputs{Text: testBase, Count: 7}
	})))
	if err != nil {
		t.Fatal(err)
	}
	if err := cnt.Provide(godi.CollectDependencies(godi.Replace(func() string { return "x" }))); err == nil {
		t.Fatal("partial replacement accepted")
	}
	if err := cnt.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := cnt.Invoke(func(s string, n int) {
		if s != testBase || n != 7 {
			t.Fatalf("original outputs lost: %q %d", s, n)
		}
	}); err != nil {
		t.Fatal(err)
	}
}

type reviewGrouped struct {
	dig.Out
	Text  string
	Items []int `group:"items,flatten"`
}

type reviewNestedGroup struct {
	dig.Out
	Nested reviewGrouped
}

func TestReplaceDigOutGroupRejected(t *testing.T) {
	t.Parallel()
	for _, dep := range []godi.Dependency{
		godi.Replace(func() reviewGrouped { return reviewGrouped{} }),
		godi.Replace(func() reviewNestedGroup { return reviewNestedGroup{} }),
	} {
		if _, err := godi.NewContainer(godi.WithDependencies(godi.CollectDependencies(dep))); err == nil ||
			!strings.Contains(err.Error(), "group") {
			t.Fatalf("expected group error, got %v", err)
		}
	}
	// Even replacing every ordinary output would silently lose the group output.
	_, err := godi.NewContainer(godi.WithDependencies(godi.CollectDependencies(
		godi.NewDependency(func() reviewGrouped { return reviewGrouped{} }),
		godi.Replace(func() string { return "x" }),
	)))
	if err == nil || !strings.Contains(err.Error(), "partial replace") {
		t.Fatalf("expected partial replace error, got %v", err)
	}
}

func TestModuleNamesValidated(t *testing.T) {
	t.Parallel()
	for _, names := range [][]string{{""}, {reviewRootScope}, {"storage", "storage"}} {
		modules := make([]godi.Module, 0, len(names))
		for _, name := range names {
			modules = append(modules, godi.NewModule(name, godi.CollectDependencies()))
		}
		if _, err := godi.NewContainer(godi.WithModules(modules...)); err == nil {
			t.Fatalf("accepted module names %v", names)
		}
	}
}

type ReviewInner struct {
	dig.Out
	Value int `name:"count"`
}

type reviewOuter struct {
	dig.Out
	Inner ReviewInner
	Items []string `group:"items,flatten"`
}

type reviewEmbedded struct {
	ReviewInner
}

func TestNestedDigOutAgreement(t *testing.T) {
	t.Parallel()
	calls := 0
	dep := godi.NewDependency(func() reviewOuter {
		calls++
		return reviewOuter{Inner: ReviewInner{Value: 7}, Items: []string{"a", "b"}}
	})
	cnt, err := godi.NewContainer(godi.WithModules(godi.NewModule("values", godi.CollectDependencies(dep))),
		godi.WithDependencies(godi.CollectDependencies(godi.Decorate(func(in struct {
			dig.In
			Value int `name:"count"`
		},
		) reviewEmbedded {
			return reviewEmbedded{ReviewInner: ReviewInner{Value: in.Value + 1}}
		}))))
	if err != nil {
		t.Fatal(err)
	}
	if err := cnt.Validate(); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("Validate ran the constructor")
	}
	if err := cnt.Invoke(func(in struct {
		dig.In
		Value int      `name:"count"`
		Items []string `group:"items"`
	},
	) {
		if in.Value != 8 || len(in.Items) != 2 {
			t.Fatalf("nested outputs lost: %+v", in)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("constructor called %d times", calls)
	}
	if got := dep.ExposedTypes(); !reflect.DeepEqual(got, []reflect.Type{reflect.TypeFor[int](), reflect.TypeFor[string]()}) {
		t.Fatalf("incorrect exposed types: %v", got)
	}
	graph := cnt.Graph()
	for _, node := range graph.Providers {
		for _, token := range node.Provides {
			if token.Type != "int" && token.Type != "string" {
				t.Fatalf("incorrect graph output: %+v", token)
			}
		}
	}
	for _, edge := range graph.Edges {
		if edge.Missing {
			t.Fatalf("missing nested output: %+v", edge)
		}
	}
}

func TestInvalidDependencyIsDiagnostic(t *testing.T) {
	t.Parallel()
	var nilConstructor func() string
	for _, dep := range []godi.Dependency{
		{},
		godi.NewDependency(nilConstructor), godi.Replace(nilConstructor),
		godi.Decorate(nilConstructor),
	} {
		if dep.Error() == nil || dep.Type() != nil || len(dep.ExposedTypes()) != 0 || dep.IsRunnable() {
			t.Fatal("invalid dependency has valid metadata")
		}
		if _, err := godi.NewContainer(godi.WithDependencies(godi.CollectDependencies(dep))); err == nil {
			t.Fatal("invalid dependency accepted")
		}
		_ = godi.BuildGraph(godi.CollectDependencies(dep)).DOT()
		_ = godi.DetectOverrides(godi.CollectDependencies(dep))
	}
}

func TestGraphModuleProviderIDs(t *testing.T) {
	t.Parallel()
	cnt, err := godi.NewContainer(godi.WithModules(
		godi.NewModule("a", godi.NewSingleDependency(func() string { return "a" }, godi.WithKey("config"))),
		godi.NewModule("b", godi.NewSingleDependency(func() int { return 7 }, godi.WithKey("config"))),
	), godi.WithDependencies(godi.NewSingleDependency(func(s string, n int) bool { return s != "" && n > 0 })))
	if err != nil {
		t.Fatal(err)
	}
	g := cnt.Graph()
	ids := make(map[string]bool)
	for _, node := range g.Providers {
		if ids[node.ID] {
			t.Fatalf("duplicate ID %q", node.ID)
		}
		ids[node.ID] = true
	}
	if len(ids) != 3 || len(g.Edges) != 2 || g.Edges[0].To == g.Edges[1].To {
		t.Fatalf("merged provider edges: %+v", g)
	}
	for _, edge := range g.Edges {
		if edge.Missing || !ids[edge.To] {
			t.Fatalf("invalid edge: %+v", edge)
		}
	}
}

func TestDetectOverridesReplaceFirst(t *testing.T) {
	t.Parallel()
	deps := godi.CollectDependencies(
		godi.Replace(func() string { return "next" }),
		godi.NewDependency(func() string { return "previous" }),
	)
	got := godi.DetectOverrides(deps)
	if len(got) != 1 || got[0].Previous.Index != 1 || got[0].Next.Index != 0 {
		t.Fatalf("incorrect override: %+v", got)
	}
}
