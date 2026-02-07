package godi_test

import (
	"testing"

	"github.com/assurrussa/godi"
)

func TestDetectOverridesReportsReplace(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		godi.NewDependency(func() string { return "base" }, godi.WithName("n")), //nolint:goconst // it's tests
		godi.Replace(func() string { return "override" }, godi.WithName("n")),   //nolint:goconst // it's tests
	)

	overrides := godi.DetectOverrides(deps)
	if len(overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(overrides))
	}
	if overrides[0].Key != "string[name=n]" {
		t.Fatalf("unexpected override key: %q", overrides[0].Key)
	}
	//nolint:goconst // it's tests
	if overrides[0].Previous.Type != "string" || overrides[0].Next.Type != "string" {
		t.Fatalf("unexpected override types: prev=%q next=%q", overrides[0].Previous.Type, overrides[0].Next.Type)
	}
	if overrides[0].Previous.Name != "n" || overrides[0].Next.Name != "n" {
		t.Fatalf("unexpected override names: prev=%q next=%q", overrides[0].Previous.Name, overrides[0].Next.Name)
	}
}
