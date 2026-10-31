package godi_test

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.uber.org/dig"

	"github.com/assurrussa/godi"
)

func TestWithConstructorsMixedRegistrationAndLazyResolve(t *testing.T) {
	t.Parallel()

	calls := 0
	buffer := bytes.NewBufferString("world")
	cnt := newAPIContainer(t,
		godi.WithConstructors(),
		godi.WithDependencies(godi.NewSingleDependency(func() string { return "hello " })),
		godi.WithConstructors(godi.NewDependency(func() *bytes.Buffer {
			calls++
			return buffer
		}).As[io.Reader]()),
		godi.WithConstructors(func(prefix string, reader io.Reader) ([]byte, error) {
			target, err := io.ReadAll(reader)
			if err != nil {
				return nil, err
			}
			return append([]byte(prefix), target...), nil
		}),
	)
	if err := cnt.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if calls != 0 {
		t.Fatalf("registration or Validate ran constructor %d times", calls)
	}

	got, err := cnt.Resolve[[]byte]()
	if err != nil || string(got) != "hello world" {
		t.Fatalf("Resolve = %q, %v", got, err)
	}
	reader, err := cnt.Resolve[io.Reader]()
	if err != nil || reader != buffer || calls != 1 {
		t.Fatalf("interface identity or lazy caching lost: reader=%v calls=%d err=%v", reader, calls, err)
	}
}

func TestWithConstructorsPreservesDependencyOptionsAndKinds(t *testing.T) {
	t.Parallel()

	cnt := newAPIContainer(t,
		godi.WithConstructors(func() int { return 1 }),
		godi.WithConstructors(
			godi.Replace(func() int { return 2 }),
			godi.Decorate(func(value int) int { return value + 1 }),
			godi.NewDependency(func() string { return testBase }, godi.WithName(testNameN), godi.WithKey("named")),
			godi.NewDependency(func() string { return "item" }, godi.WithGroup("items")),
		),
	)
	type input struct {
		dig.In
		Value int
		Named string   `name:"n"`
		Items []string `group:"items"`
	}
	got, err := cnt.Resolve[input]()
	if err != nil || got.Value != 3 || got.Named != testBase || !slices.Equal(got.Items, []string{"item"}) {
		t.Fatalf("configured dependencies lost: %+v, %v", got, err)
	}
	foundKey := false
	for _, node := range cnt.Graph().Providers {
		foundKey = foundKey || node.Key == "named"
	}
	if !foundKey {
		t.Fatal("configured dependency lost graph metadata")
	}
}

func TestWithConstructorsCapturesInputList(t *testing.T) {
	t.Parallel()

	constructors := []any{func() int { return 7 }}
	option := godi.WithConstructors(constructors...)
	constructors[0] = nil
	cnt := newAPIContainer(t, option)
	got, err := cnt.Resolve[int]()
	if err != nil || got != 7 {
		t.Fatalf("caller changed captured constructor list: %d, %v", got, err)
	}
}

func TestWithConstructorsRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	var nilConstructor func() int
	dep := godi.NewDependency(func() int { return 1 })
	tests := map[string]any{
		"nil":              nil,
		"typed nil":        nilConstructor,
		"non-function":     42,
		"dependency ptr":   &dep,
		"zero dependency":  godi.Dependency{},
		"no output":        func() {},
		"only error":       func() error { return nil },
		"invalid prepared": godi.NewDependency(nil),
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := godi.NewContainer(godi.WithConstructors(input))
			if err == nil {
				t.Fatal("expected invalid constructor error")
			}
		})
	}
}

func TestDependencyAsChainsInterfaces(t *testing.T) {
	t.Parallel()

	buffer := bytes.NewBufferString("value")
	cnt := newAPIContainer(t, godi.WithConstructors(
		godi.NewDependency(func() *bytes.Buffer { return buffer }).As[io.Reader]().As[io.Writer](),
	))
	reader, err := cnt.Resolve[io.Reader]()
	if err != nil || reader != buffer {
		t.Fatalf("Resolve reader: %v, %v", reader, err)
	}
	writer, err := cnt.Resolve[io.Writer]()
	if err != nil || writer != buffer {
		t.Fatalf("Resolve writer: %v, %v", writer, err)
	}
	concrete, err := cnt.Resolve[*bytes.Buffer]()
	if err == nil || concrete != nil {
		t.Fatalf("concrete type must not be exposed: %v, %v", concrete, err)
	}
}

func TestDependencyAsLeavesSiblingChainsIndependent(t *testing.T) {
	t.Parallel()

	// Three appends leave capacity for a fourth entry in the shared backing
	// array if As forgets to copy the slice before appending.
	base := godi.NewDependency(func() *bytes.Buffer { return new(bytes.Buffer) }).
		As[io.Reader]().As[io.Writer]().As[io.ByteReader]()
	left := base.As[io.ByteWriter]()
	right := base.As[io.StringWriter]()
	common := []reflect.Type{reflect.TypeFor[io.Reader](), reflect.TypeFor[io.Writer](), reflect.TypeFor[io.ByteReader]()}
	tests := []struct {
		name string
		dep  godi.Dependency
		want []reflect.Type
	}{
		{name: "base", dep: base, want: common},
		{name: "left", dep: left, want: append(slices.Clone(common), reflect.TypeFor[io.ByteWriter]())},
		{name: "right", dep: right, want: append(slices.Clone(common), reflect.TypeFor[io.StringWriter]())},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.dep.ExposedTypes(); !slices.Equal(got, test.want) {
				t.Fatalf("As changed another dependency: got %v, want %v", got, test.want)
			}
			cnt := newAPIContainer(t, godi.WithConstructors(test.dep))
			if err := cnt.Validate(); err != nil {
				t.Fatalf("Validate: %v", err)
			}
		})
	}
}

func TestDependencyAsPreservesOriginalAndErrors(t *testing.T) {
	t.Parallel()

	base := godi.NewDependency(func() *bytes.Buffer { return new(bytes.Buffer) })
	invalid := base.As[int]()
	if invalid.Error() == nil || base.Error() != nil || !slices.Equal(base.ExposedTypes(), []reflect.Type{base.Type()}) {
		t.Fatal("As must reject concrete targets without changing the original dependency")
	}
	broken := godi.NewDependency(nil)
	originalErr := broken.Error()
	chained := broken.As[int]().As[io.Reader]()
	if !errors.Is(chained.Error(), originalErr) {
		t.Fatalf("As replaced the first error: %v", chained.Error())
	}
	zero := (godi.Dependency{}).As[io.Reader]()
	if zero.Error() == nil {
		t.Fatal("As must not make a zero Dependency valid")
	}
}

func TestDependencyAsRejectsUnsupportedBindings(t *testing.T) {
	t.Parallel()

	type output struct {
		dig.Out
		Reader io.Reader
	}
	tests := map[string]godi.Dependency{
		"concrete target": godi.NewDependency(func() int { return 1 }).As[int](),
		"unimplemented":   godi.NewDependency(func() int { return 1 }).As[io.Reader](),
		"result object": godi.NewDependency(func() output {
			return output{Reader: new(bytes.Buffer)}
		}).As[io.Reader](),
		"decorator": godi.Decorate(func(reader io.Reader) io.Reader { return reader }).As[io.Reader](),
	}
	for name, dep := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := godi.NewContainer(godi.WithConstructors(dep))
			if err == nil {
				t.Fatal("expected invalid interface binding error")
			}
		})
	}
}

func TestResolvePreservesPointerIdentity(t *testing.T) {
	t.Parallel()

	calls := 0
	buffer := new(bytes.Buffer)
	cnt := newAPIContainer(t, godi.WithConstructors(func() *bytes.Buffer {
		calls++
		return buffer
	}))
	for range 2 {
		got, err := cnt.Resolve[*bytes.Buffer]()
		if err != nil || got != buffer || calls != 1 {
			t.Fatalf("Resolve changed instance: %v, calls=%d, err=%v", got, calls, err)
		}
	}
	assertProvideAfterResolveFails(t, cnt)
}

func TestResolveReturnsOriginalConstructorError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("constructor failed")
	cnt := newAPIContainer(t, godi.WithConstructors(func() (int, error) { return 9, wantErr }))
	got, err := cnt.Resolve[int]()
	if got != 0 || !errors.Is(err, wantErr) {
		t.Fatalf("Resolve = %d, %v; want zero and original error", got, err)
	}
	assertProvideAfterResolveFails(t, cnt)
}

func TestResolveMissingTypeAndConstructorPanic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []godi.ContainerOption
		text string
	}{
		{name: "missing", text: "missing"},
		{
			name: "panic",
			opts: []godi.ContainerOption{godi.WithConstructors(func() int { panic("constructor exploded") })},
			text: "constructor exploded",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cnt := newAPIContainer(t, test.opts...)
			got, err := cnt.Resolve[int]()
			if got != 0 || err == nil || !strings.Contains(err.Error(), test.text) {
				t.Fatalf("Resolve = %d, %v; want zero and %q error", got, err, test.text)
			}
			assertProvideAfterResolveFails(t, cnt)
		})
	}
}

func newAPIContainer(t *testing.T, opts ...godi.ContainerOption) *godi.Container {
	t.Helper()
	cnt, err := godi.NewContainer(opts...)
	if err != nil {
		t.Fatalf("NewContainer: %v", err)
	}
	return cnt
}

func assertProvideAfterResolveFails(t *testing.T, cnt *godi.Container) {
	t.Helper()
	err := cnt.Provide(godi.NewSingleDependency(func() bool { return true }))
	if err == nil || !strings.Contains(err.Error(), "started") {
		t.Fatalf("Provide after Resolve must fail: %v", err)
	}
}
