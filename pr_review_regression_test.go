package godi_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.uber.org/dig"

	"github.com/assurrussa/godi"
)

const (
	scopePrivate  = "private"
	panicRollback = "rollback"
	panicAtStart  = "start"
	panicStartA   = "start-first"
	panicStartB   = "start-second"
	panicStopA    = "stop-first"
	panicStopB    = "stop-second"
)

func TestCompleteReplacementAcrossScopeBoundaries(t *testing.T) {
	t.Parallel()
	for _, otherModule := range []bool{false, true} {
		for _, private := range []bool{false, true} {
			t.Run(fmt.Sprintf("module=%v/private=%v", otherModule, private), func(t *testing.T) {
				t.Parallel()
				checkCompleteReplacementAcrossScopes(t, otherModule, private)
			})
		}
	}
}

func checkCompleteReplacementAcrossScopes(t *testing.T, otherModule, private bool) {
	t.Helper()
	calls := 0
	var depOptions []godi.DependencyOption
	if private {
		depOptions = append(depOptions, godi.Private())
	}
	module := godi.NewModule("original", godi.CollectDependencies(
		godi.NewDependency(func() reviewOutputs { calls++; return reviewOutputs{} }, depOptions...),
		godi.Replace(func() string { return reviewReplacement }, depOptions...),
	))
	count := godi.CollectDependencies(godi.Replace(func() int { return 9 }))
	opts := []godi.ContainerOption{godi.WithModules(module)}
	if otherModule {
		opts = append(opts, godi.WithModules(godi.NewModule("replacement", count)))
	} else {
		opts = append(opts, godi.WithDependencies(count))
	}
	cnt, err := godi.NewContainer(opts...)
	if private {
		if err == nil || !strings.Contains(err.Error(), "partial replace") {
			t.Fatalf("private outputs must be replaced locally, got %v", err)
		}
		return
	}
	if err != nil {
		t.Fatalf("complete replacement (other module=%v): %v", otherModule, err)
	}
	if err := cnt.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := cnt.Invoke(func(s string, n int) {
		if s != reviewReplacement || n != 9 {
			t.Fatalf("incorrect replacements: %q %d", s, n)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("original constructor ran %d times", calls)
	}
	for scope, graph := range cnt.GraphModules() {
		if len(graph.Providers) != 2 {
			t.Fatalf("scope %s retained replaced constructor: %+v", scope, graph.Providers)
		}
	}
}

type scopedConsumer struct {
	text  string
	items []int
}

type scopedConsumerIn struct {
	dig.In
	Text  string
	Items []int `group:"items"`
}

func TestModuleGraphPrivateGroupVisibility(t *testing.T) {
	t.Parallel()
	for _, mixedOutputs := range []bool{false, true} {
		cnt := newScopedGraphContainer(t, mixedOutputs)
		if err := cnt.Validate(); err != nil {
			t.Fatal(err)
		}
		if err := cnt.Invoke(func(c *scopedConsumer, s string, n int) {
			if c.text != scopePrivate || len(c.items) != 2 || s != "global" || n != 7 {
				t.Fatalf("incorrect scope visibility: %+v, %q %d", c, s, n)
			}
		}); err != nil {
			t.Fatal(err)
		}
		assertPrivateGraphVisibility(t, cnt.GraphModules()["scope"])
	}
}

func assertPrivateGraphVisibility(t *testing.T, graph godi.Graph) {
	t.Helper()
	ids := make(map[string]string)
	for _, node := range graph.Providers {
		ids[node.Key] = node.ID
		if node.Kind == "replace" {
			t.Fatalf("scope visibility synthesized Replace: %+v", node)
		}
	}
	textEdges, groupEdges := 0, 0
	for _, edge := range graph.Edges {
		if edge.From != ids["consumer"] {
			continue
		}
		if edge.Type == testTypeString {
			textEdges++
			if edge.To != ids["local"] || edge.Missing {
				t.Fatalf("wrong private string edge: %+v", edge)
			}
		}
		if edge.Group == "items" {
			groupEdges++
			if edge.Missing {
				t.Fatalf("missing group edge: %+v", edge)
			}
		}
	}
	if textEdges != 1 || groupEdges != 2 {
		t.Fatalf("expected one private string and two additive group edges, got %d and %d", textEdges, groupEdges)
	}
	if ids["global"] == "" {
		t.Fatal("global provider with an unshadowed int output was dropped")
	}
}

func newScopedGraphContainer(t *testing.T, mixedOutputs bool) *godi.Container {
	t.Helper()
	local := []godi.Dependency{}
	if mixedOutputs {
		local = append(local, godi.NewDependency(func() reviewGrouped {
			return reviewGrouped{Text: scopePrivate, Items: []int{2}}
		}, godi.Private(), godi.WithKey("local")))
	} else {
		local = append(local,
			godi.NewDependency(func() string { return scopePrivate }, godi.Private(), godi.WithKey("local")),
			godi.NewDependency(func() struct {
				dig.Out
				Item int `group:"items"`
			} {
				return struct {
					dig.Out
					Item int `group:"items"`
				}{Item: 2}
			}, godi.Private(), godi.WithKey("local-group")),
		)
	}
	local = append(local, godi.NewDependency(func(in scopedConsumerIn) *scopedConsumer {
		return &scopedConsumer{text: in.Text, items: in.Items}
	}, godi.WithKey("consumer")))
	cnt, err := godi.NewContainer(godi.WithDependencies(godi.CollectDependencies(
		godi.NewDependency(func() reviewOutputs { return reviewOutputs{Text: "global", Count: 7} }, godi.WithKey("global")),
		godi.NewDependency(func() int { return 1 }, godi.WithGroup("items"), godi.WithKey("global-group")),
	)), godi.WithModules(godi.NewModule("scope", godi.CollectDependencies(local...))))
	if err != nil {
		t.Fatal(err)
	}
	return cnt
}

func TestLifecycleHookPanicsFinishTransitions(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{panicAtStart, panicRollback, "stop"} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()
			checkLifecycleHookPanic(t, phase)
		})
	}
}

func checkLifecycleHookPanic(t *testing.T, phase string) {
	t.Helper()
	l := godi.NewLifecycle()
	var calls []string
	panicErr := errors.New("hook exploded")
	startErr := errors.New("startup failed")
	l.Append(godi.Hook{
		OnStart: func(context.Context) error { calls = append(calls, panicStartA); return nil },
		OnStop:  func(context.Context) error { calls = append(calls, panicStopA); return nil },
	})
	l.Append(godi.Hook{
		OnStart: func(context.Context) error {
			calls = append(calls, panicStartB)
			if phase == panicAtStart {
				panic(panicErr)
			}
			return nil
		},
		OnStop: func(context.Context) error { calls = append(calls, panicStopB); panic(panicErr) },
	})
	l.Append(godi.Hook{OnStart: func(context.Context) error {
		if phase == panicRollback {
			return startErr
		}
		return nil
	}})
	// dig catches any escaping panic, reproducing real container startup.
	cnt, err := godi.NewContainer()
	if err != nil {
		t.Fatal(err)
	}
	err = cnt.Invoke(func() error { return l.Start(context.Background()) })
	if phase == "stop" {
		if err != nil {
			t.Fatal(err)
		}
		err = cnt.Invoke(func() error { return l.Stop(context.Background()) })
	}
	if !errors.Is(err, panicErr) || !strings.Contains(err.Error(), "panicked") {
		t.Errorf("panic was not returned as a diagnostic error: %v", err)
	}
	var panicDiagnostic *godi.HookPanicError
	if !errors.As(err, &panicDiagnostic) {
		t.Fatalf("missing HookPanicError: %v", err)
	}
	wantPhase := "OnStop"
	if phase == panicAtStart {
		wantPhase = "OnStart"
	}
	if panicDiagnostic.Phase != wantPhase || panicDiagnostic.Index != 1 || !errors.Is(panicDiagnostic, panicErr) ||
		!strings.Contains(string(panicDiagnostic.Stack), "TestLifecycleHookPanicsFinishTransitions") {
		t.Fatalf("incomplete panic diagnostic: %+v", panicDiagnostic)
	}
	if phase == panicRollback && !errors.Is(err, startErr) {
		t.Errorf("lost startup error: %v", err)
	}
	stopErr := l.Stop(context.Background())
	if phase == panicAtStart {
		if stopErr != nil {
			t.Errorf("shutdown failed after start panic: %v", stopErr)
		}
	} else if !errors.Is(stopErr, panicErr) {
		t.Errorf("shutdown lost cleanup panic: %v", stopErr)
	}
	want := []string{panicStartA, panicStartB, panicStopB, panicStopA}
	if phase == panicAtStart {
		want = []string{panicStartA, panicStartB, panicStopA}
	}
	if !reflect.DeepEqual(calls, want) {
		t.Errorf("cleanup interrupted or repeated: got %v, want %v", calls, want)
	}
}

func TestLifecycleStopOnlyPanicKeepsRegistrationIndex(t *testing.T) {
	t.Parallel()
	l := godi.NewLifecycle()
	stops := 0
	l.Append(godi.Hook{OnStart: func(context.Context) error { return nil }})
	l.Append(godi.Hook{OnStop: func(context.Context) error { stops++; return nil }})
	l.Append(godi.Hook{OnStop: func(context.Context) error { panic("stop-only panic") }})
	for range 2 {
		err := l.Stop(context.Background())
		var diagnostic *godi.HookPanicError
		if !errors.As(err, &diagnostic) || diagnostic.Index != 2 || diagnostic.Phase != "OnStop" ||
			diagnostic.Value != "stop-only panic" || diagnostic.Unwrap() != nil {
			t.Fatalf("incorrect stop-only panic: %v", err)
		}
	}
	if stops != 1 {
		t.Fatalf("stop-only cleanup ran %d times", stops)
	}
}

func TestWithMatchingsValidatedWithoutProvideDependencies(t *testing.T) {
	t.Parallel()

	tests := map[string][]godi.ContainerOption{
		"empty container invalid type": {
			godi.WithMatchings("not-a-matching"),
		},
		"empty container invalid Matching": {
			godi.WithMatchings(godi.NewMatching(123, new(io.Reader))),
		},
		"decorate only invalid Matching": {
			godi.WithMatchings("not-a-matching"),
			godi.WithConstructors(godi.Decorate(func(v int) int { return v })),
		},
	}

	for name, opts := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := godi.NewContainer(opts...); err == nil {
				t.Fatal("expected error for invalid WithMatchings")
			}
		})
	}
}

func TestWithMatchingsDoesNotMutateSharedDependencySlice(t *testing.T) {
	t.Parallel()

	base := godi.NewDependency(func() *bytes.Buffer { return new(bytes.Buffer) }).
		As[io.Reader]().As[io.Writer]().As[io.ByteReader]()
	wantBase := []reflect.Type{
		reflect.TypeFor[io.Reader](),
		reflect.TypeFor[io.Writer](),
		reflect.TypeFor[io.ByteReader](),
	}

	cnt, err := godi.NewContainer(
		godi.WithMatchings(new(io.ByteWriter)),
		godi.WithConstructors(base),
	)
	if err != nil {
		t.Fatalf("NewContainer: %v", err)
	}
	if _, err := cnt.Resolve[io.ByteWriter](); err != nil {
		t.Fatalf("Resolve io.ByteWriter: %v", err)
	}

	sibling := base.As[io.StringWriter]()
	if !slices.Equal(base.ExposedTypes(), wantBase) {
		t.Fatalf("WithMatchings mutated base dependency: got %v, want %v", base.ExposedTypes(), wantBase)
	}
	wantSibling := append(slices.Clone(wantBase), reflect.TypeFor[io.StringWriter]())
	if !slices.Equal(sibling.ExposedTypes(), wantSibling) {
		t.Fatalf("WithMatchings polluted sibling chain: got %v, want %v", sibling.ExposedTypes(), wantSibling)
	}
}

type NestedDigInInner struct {
	dig.In
	Name string `name:"service"`
}

type nestedDigInOuter struct {
	NestedDigInInner
	Opt godi.Optional[int]
}

func TestGraphNestedDigInAndDeterministicMissingDOT(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		godi.NewDependency(func(in nestedDigInOuter) bool {
			_, ok := in.Opt.Get()
			return in.Name != "" && ok
		}, godi.WithKey("consumer")),
	)

	g := godi.BuildGraph(deps)
	if len(g.Providers) != 1 || len(g.Providers[0].Requires) != 2 {
		t.Fatalf("expected 2 required tokens from nested dig.In, got %+v", g.Providers)
	}

	reqs := g.Providers[0].Requires
	if reqs[0].Type != testTypeString || reqs[0].Name != "service" || reqs[0].Optional {
		t.Fatalf("unexpected first require token: %+v", reqs[0])
	}
	if reqs[1].Type != "*int" || !reqs[1].Optional {
		t.Fatalf("unexpected optional require token: %+v", reqs[1])
	}

	firstDOT := g.DOT()
	for range 10 {
		if got := g.DOT(); got != firstDOT {
			t.Fatalf("non-deterministic DOT output:\ngot:\n%s\nwant:\n%s", got, firstDOT)
		}
	}
}

func TestModuleOrderDeterminismInValidateAndGroups(t *testing.T) {
	t.Parallel()

	modA := godi.NewModule("a", godi.NewSingleDependency(func(_ int) string { return "a" }))
	modB := godi.NewModule("b", godi.NewSingleDependency(func(_ bool) float64 { return 1 }))

	cnt, err := godi.NewContainer(godi.WithModules(modA, modB))
	if err != nil {
		t.Fatalf("NewContainer: %v", err)
	}
	for range 10 {
		err := cnt.Validate()
		if err == nil || !strings.HasPrefix(err.Error(), "module a:") {
			t.Fatalf("expected Validate to fail on module a first, got %v", err)
		}
	}
}
