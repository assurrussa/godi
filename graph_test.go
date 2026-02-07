package godi_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/assurrussa/godi"
)

const (
	kindDecorate = "decorate"
	kindReplace  = "replace"
)

func idxFromProviderID(id string) (int, bool) {
	i := strings.LastIndex(id, "#")
	if i < 0 || i == len(id)-1 {
		return 0, false
	}
	n, err := strconv.Atoi(id[i+1:])
	if err != nil {
		return 0, false
	}
	return n, true
}

func TestBuildGraphDecoratorChainEdges(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		godi.NewDependency(func() string { return "base" }),
		godi.Decorate(func(s string) string { return s + "1" }),
		godi.Decorate(func(s string) string { return s + "2" }),
	)

	g := godi.BuildGraph(deps)

	nodesByIdx := map[int]godi.ProviderNode{}
	for _, n := range g.Providers {
		idx, ok := idxFromProviderID(n.ID)
		if !ok {
			continue
		}
		nodesByIdx[idx] = n
	}

	base, ok := nodesByIdx[0]
	if !ok {
		t.Fatal("expected base provider node with idx 0")
	}
	d1, ok := nodesByIdx[1]
	if !ok || d1.Kind != kindDecorate {
		t.Fatalf("expected decorator node idx 1, got %+v", d1)
	}
	d2, ok := nodesByIdx[2]
	if !ok || d2.Kind != kindDecorate {
		t.Fatalf("expected decorator node idx 2, got %+v", d2)
	}

	var edgeD1, edgeD2 godi.ProviderEdge
	var hasD1, hasD2 bool
	for i := range g.Edges {
		e := g.Edges[i]
		if e.Type != "string" || e.Name != "" || e.Group != "" || e.Missing {
			continue
		}
		if e.From == d1.ID {
			edgeD1 = e
			hasD1 = true
		}
		if e.From == d2.ID {
			edgeD2 = e
			hasD2 = true
		}
	}

	if !hasD1 || edgeD1.To != base.ID {
		t.Fatalf("expected decorator1 to depend on base, got %+v", edgeD1)
	}
	if !hasD2 || edgeD2.To != d1.ID {
		t.Fatalf("expected decorator2 to depend on decorator1, got %+v", edgeD2)
	}
}

func TestBuildGraphMissingDependencyInDOT(t *testing.T) {
	t.Parallel()

	deps := godi.NewSingleDependency(func(_ int) string { return "x" })
	g := godi.BuildGraph(deps)
	dot := g.DOT()

	if !strings.Contains(dot, "missing:int") {
		t.Fatalf("expected DOT to contain missing int node, got:\n%s", dot)
	}
}

func TestGraphModulesShowsPrivateProvidersInModuleGraphOnly(t *testing.T) {
	t.Parallel()

	module := godi.Module{
		Name: "m",
		Dependencies: godi.CollectDependencies(
			godi.NewDependency(func() string { return "secret" }, godi.Private()),
		),
	}

	cnt, err := godi.NewContainer(godi.WithDependencies(
		godi.NewSingleDependency(func() string { return "root" }),
	), godi.WithModules(module))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	graphs := cnt.GraphModules()
	root := graphs["root"]
	mod := graphs["m"]

	var rootStrings int
	for _, p := range root.Providers {
		if p.Type == "string" && p.Kind != kindDecorate {
			rootStrings++
		}
	}
	if rootStrings != 1 {
		t.Fatalf("expected root graph to have 1 string provider, got %d", rootStrings)
	}

	var modStrings, modReplaces int
	for _, p := range mod.Providers {
		if p.Type == "string" && p.Kind != kindDecorate {
			modStrings++
		}
		if p.Type == "string" && p.Kind == kindReplace {
			modReplaces++
		}
	}
	// Module graph is resolved, so the private provider wins and root provider is not shown.
	if modStrings != 1 {
		t.Fatalf("expected module graph to have 1 resolved string provider, got %d", modStrings)
	}
	if modReplaces != 1 {
		t.Fatalf("expected module graph to mark private provider as replace, got %d", modReplaces)
	}
}
