package godi_test

import (
	"bytes"
	"io"
	"sort"
	"testing"

	"go.uber.org/dig"

	"github.com/assurrussa/godi"
)

type testReaderImpl struct{}

func (testReaderImpl) Read([]byte) (int, error) { return 0, nil }

func TestContainerDuplicateProvideFails(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		godi.NewDependency(func() string { return "first" }),
		godi.NewDependency(func() string { return "second" }),
	)

	if _, err := godi.NewContainer(godi.WithDependencies(deps)); err == nil {
		t.Fatal("expected error for duplicate provider, got nil")
	}
}

func TestContainerReplaceOverridesProvide(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		godi.NewDependency(func() string { return testBase }),
		godi.Replace(func() string { return testOverride }),
	)

	cnt, err := godi.NewContainer(godi.WithDependencies(deps))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	var got string
	if err := cnt.Invoke(func(v string) { got = v }); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if got != testOverride {
		t.Fatalf("expected replacement to win, got %q", got)
	}
}

func TestContainerReplaceWithoutBase(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		godi.Replace(func() string { return "only" }),
	)

	cnt, err := godi.NewContainer(godi.WithDependencies(deps))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	var got string
	if err := cnt.Invoke(func(v string) { got = v }); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if got != "only" {
		t.Fatalf("expected replace to behave like provide, got %q", got)
	}
}

func TestContainerDecorateRequiresBase(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		godi.Decorate(func(s string) string { return s + "!" }),
	)

	if _, err := godi.NewContainer(godi.WithDependencies(deps)); err == nil {
		t.Fatal("expected error for decorate without base provider, got nil")
	}
}

func TestContainerDecorateSuccess(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		godi.NewDependency(func() string { return testBase }),
		godi.Decorate(func(s string) string { return s + "!" }),
	)

	cnt, err := godi.NewContainer(godi.WithDependencies(deps))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	var got string
	if err := cnt.Invoke(func(v string) { got = v }); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if got != "base!" {
		t.Fatalf("expected decorated value, got %q", got)
	}
}

func TestGroupMultiBind(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		godi.NewDependency(func() string { return "a" }, godi.WithGroup("items")),
		godi.NewDependency(func() string { return "b" }, godi.WithGroup("items")),
	)

	cnt, err := godi.NewContainer(godi.WithDependencies(deps))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	var got []string
	if err := cnt.Invoke(func(in struct {
		dig.In
		Items []string `group:"items"`
	},
	) {
		got = in.Items
	}); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
}

func TestModulePrivateVisibility(t *testing.T) {
	t.Parallel()

	module := godi.Module{
		Name: "test",
		Dependencies: godi.CollectDependencies(
			godi.NewDependency(func() string { return "secret" }, godi.Private()),
			godi.NewDependency(func(s string) int { return len(s) }),
		),
	}

	cnt, err := godi.NewContainer(godi.WithModules(module))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	var got int
	if err := cnt.Invoke(func(v int) { got = v }); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if got != len("secret") {
		t.Fatalf("expected %d, got %d", len("secret"), got)
	}

	if err := cnt.Invoke(func(_ string) {}); err == nil {
		t.Fatal("expected private dependency to be hidden from root")
	}
}

func TestWithMatchInvalidReturnsError(t *testing.T) {
	t.Parallel()

	_, err := godi.NewContainer(godi.WithDependencies(
		godi.CollectDependencies(
			godi.NewDependency(func() string { return "value" }, godi.WithMatch("not-an-interface-pointer")),
		),
	))
	if err == nil {
		t.Fatal("expected error for invalid WithMatch, got nil")
	}
}

func TestProvideAfterInvokeFails(t *testing.T) {
	t.Parallel()

	cnt, err := godi.NewContainer(godi.WithDependencies(
		godi.CollectDependencies(godi.NewDependency(func() string { return "value" })),
	))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	if err := cnt.Invoke(func(_ string) {}); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}

	if err := cnt.Provide(godi.CollectDependencies(
		godi.NewDependency(func() int { return 1 }),
	)); err == nil {
		t.Fatal("expected error when providing after invoke, got nil")
	}
}

func TestValidateDryRun(t *testing.T) {
	t.Parallel()

	cnt, err := godi.NewContainer(godi.WithDependencies(
		godi.CollectDependencies(
			godi.NewDependency(func() string { return "ok" }),
			godi.NewDependency(func(_ int) (bool, error) { return false, nil }),
		),
	))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	if err := cnt.Validate(); err == nil {
		t.Fatal("expected validation error for missing dependency, got nil")
	}
}

func TestValidateWithMatch(t *testing.T) {
	t.Parallel()

	cnt, err := godi.NewContainer(godi.WithDependencies(
		godi.CollectDependencies(
			godi.NewDependency(func() *bytes.Buffer { return bytes.NewBufferString("ok") }, godi.WithMatch(new(io.Reader))),
		),
	))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	if err := cnt.Validate(); err != nil {
		t.Fatalf("expected validation to pass, got %v", err)
	}
}

func TestProvideDigOutSlots(t *testing.T) {
	t.Parallel()

	type out struct {
		dig.Out
		Named string `name:"named"`
		Item  int    `group:"items"`
	}

	cnt, err := godi.NewContainer(godi.WithDependencies(
		godi.CollectDependencies(
			godi.NewDependency(func() out {
				return out{Named: "ok", Item: 7}
			}),
		),
	))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	if err := cnt.Validate(); err != nil {
		t.Fatalf("expected validation to pass, got %v", err)
	}

	var gotName string
	var gotItems []int
	if err := cnt.Invoke(func(in struct {
		dig.In
		Named string `name:"named"`
		Items []int  `group:"items"`
	},
	) {
		gotName = in.Named
		gotItems = in.Items
	}); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}

	if gotName != "ok" {
		t.Fatalf("expected named value, got %q", gotName)
	}
	if len(gotItems) != 1 || gotItems[0] != 7 {
		t.Fatalf("expected grouped value [7], got %v", gotItems)
	}
}

func TestProvideDigOutGroupFlatten(t *testing.T) {
	t.Parallel()

	type out struct {
		dig.Out
		Items []int `group:"items,flatten"`
	}

	cnt, err := godi.NewContainer(godi.WithDependencies(
		godi.NewSingleDependency(func() out { return out{Items: []int{1, 2}} }),
	))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	if err := cnt.Validate(); err != nil {
		t.Fatalf("expected validation to pass, got %v", err)
	}

	var got []int
	if err := cnt.Invoke(func(in struct {
		dig.In
		Items []int `group:"items"`
	},
	) {
		got = in.Items
	}); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}

	sort.Ints(got)
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("expected items [1 2], got %v", got)
	}
}

func TestWithMatchPointerOrigin(t *testing.T) {
	t.Parallel()

	cnt, err := godi.NewContainer(
		godi.WithDependencies(
			godi.CollectDependencies(
				godi.NewDependency(func() *testReaderImpl { return &testReaderImpl{} }),
			),
		),
		godi.WithMatchings(godi.NewMatching(new(testReaderImpl), new(io.Reader))),
	)
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	var got io.Reader
	if err := cnt.Invoke(func(r io.Reader) { got = r }); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if got == nil {
		t.Fatal("expected matched io.Reader, got nil")
	}
}

func TestNewDependencyNilConstructorReturnsError(t *testing.T) {
	t.Parallel()

	_, err := godi.NewContainer(godi.WithDependencies(
		godi.CollectDependencies(godi.NewDependency(nil)),
	))
	if err == nil {
		t.Fatal("expected error for nil constructor, got nil")
	}
}

func TestNewDependencyInvalidSignatureReturnsError(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		//nolint:nilnil // it's test
		godi.NewDependency(func() (error, error) { return nil, nil }),
	)

	_, err := godi.NewContainer(godi.WithDependencies(deps))
	if err == nil {
		t.Fatal("expected error for invalid constructor signature, got nil")
	}
}

func TestProvideIsTransactionalOnBuildFailure(t *testing.T) {
	t.Parallel()

	cnt, err := godi.NewContainer(godi.WithDependencies(
		godi.CollectDependencies(godi.NewDependency(func() string { return "ok" })),
	))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	// This provide should fail because of duplicate provider for string.
	if err := cnt.Provide(godi.CollectDependencies(godi.NewDependency(func() string { return "dup" }))); err == nil {
		t.Fatal("expected provide error, got nil")
	}

	// Container must remain usable after failed provide.
	var got string
	if err := cnt.Invoke(func(v string) { got = v }); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if got != "ok" {
		t.Fatalf("expected original dependency to remain, got %q", got)
	}
}

func TestProvideRejectsNameAndGroupTogether(t *testing.T) {
	t.Parallel()

	_, err := godi.NewContainer(godi.WithDependencies(
		godi.CollectDependencies(
			godi.NewDependency(func() string { return "x" }, godi.WithName("n"), godi.WithGroup("g")),
		),
	))
	if err == nil {
		t.Fatal("expected error for WithName+WithGroup, got nil")
	}
}

func TestProvideRejectsRunnableAndGroupTogether(t *testing.T) {
	t.Parallel()

	_, err := godi.NewContainer(godi.WithDependencies(
		godi.CollectDependencies(
			godi.NewDependency(func() godi.Runnable { return godi.Runnable{} }, godi.WithGroup("g")),
		),
	))
	if err == nil {
		t.Fatal("expected error for Runnable+WithGroup, got nil")
	}
}
