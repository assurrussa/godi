package godi_test

import (
	"testing"

	"github.com/assurrussa/godi"
)

func TestOptionalMissing(t *testing.T) {
	t.Parallel()

	cnt, err := godi.NewContainer()
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	var got string
	var ok bool
	if err := cnt.Invoke(func(o godi.Optional[string]) {
		got, ok = o.Get()
	}); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if ok {
		t.Fatalf("expected optional to be missing, got %q", got)
	}
}

func TestOptionalPresent(t *testing.T) {
	t.Parallel()

	s := "x"
	cnt, err := godi.NewContainer(godi.WithDependencies(
		// Optional[T] is modeled as an optional *T, so to make it work for T=string,
		// we provide *string.
		godi.NewSingleDependency(func() *string { return &s }),
	))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	var got string
	var ok bool
	if err := cnt.Invoke(func(o godi.Optional[string]) {
		got, ok = o.Get()
	}); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if !ok || got != "x" {
		t.Fatalf("expected optional value x, got %q (ok=%v)", got, ok)
	}
}

func TestOptionalGetPtrPreservesIdentity(t *testing.T) {
	t.Parallel()
	value := "original"
	cnt, err := godi.NewContainer(godi.WithDependencies(godi.NewSingleDependency(func() *string { return &value })))
	if err != nil {
		t.Fatal(err)
	}
	if err := cnt.Invoke(func(o godi.Optional[string]) {
		ptr, ok := o.GetPtr()
		if !ok || ptr != &value {
			t.Fatal("pointer identity lost")
		}
		*ptr = "updated"
	}); err != nil {
		t.Fatal(err)
	}
	if value != "updated" {
		t.Fatal("original object unchanged")
	}
	var missing godi.Optional[string]
	if ptr, ok := missing.GetPtr(); ok || ptr != nil {
		t.Fatal("missing pointer reported as present")
	}
}
