package godi_test

import (
	"testing"

	"github.com/assurrussa/godi"
)

func TestDetectOverridesReportsReplace(t *testing.T) {
	t.Parallel()

	deps := godi.CollectDependencies(
		godi.NewDependency(func() string { return testBase }, godi.WithName(testNameN)),
		godi.Replace(func() string { return testOverride }, godi.WithName(testNameN)),
	)

	overrides := godi.DetectOverrides(deps)
	if len(overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(overrides))
	}
	if overrides[0].Key != "string[name=n]" {
		t.Fatalf("unexpected override key: %q", overrides[0].Key)
	}
	if overrides[0].Previous.Type != testTypeString || overrides[0].Next.Type != testTypeString {
		t.Fatalf("unexpected override types: prev=%q next=%q", overrides[0].Previous.Type, overrides[0].Next.Type)
	}
	if overrides[0].Previous.Name != testNameN || overrides[0].Next.Name != testNameN {
		t.Fatalf("unexpected override names: prev=%q next=%q", overrides[0].Previous.Name, overrides[0].Next.Name)
	}
}
