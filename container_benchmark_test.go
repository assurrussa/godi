package godi_test

import (
	"fmt"
	"testing"

	"github.com/assurrussa/godi"
)

// Named slots keep fixture creation outside the measured registration work.
func benchmarkDependencies(size int) godi.Dependencies {
	deps := make([]godi.Dependency, size)
	for i := range deps {
		deps[i] = godi.NewDependency(func() int { return 1 }, godi.WithName(fmt.Sprintf("item-%d", i)))
	}
	return godi.CollectDependencies(deps...)
}

func BenchmarkContainerRegistration(b *testing.B) {
	for _, size := range []int{10, 100, 500} {
		deps := benchmarkDependencies(size)
		b.Run(fmt.Sprintf("batch/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := godi.NewContainer(godi.WithDependencies(deps)); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("individual/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				cnt, err := godi.NewContainer()
				if err != nil {
					b.Fatal(err)
				}
				for _, dep := range deps.List() {
					if err := cnt.Provide(godi.CollectDependencies(dep)); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func BenchmarkContainerDiagnostics(b *testing.B) {
	for _, size := range []int{10, 100, 500} {
		cnt, err := godi.NewContainer(godi.WithDependencies(benchmarkDependencies(size)))
		if err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("validate/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := cnt.Validate(); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("dot/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = cnt.GraphDOT()
			}
		})
	}
}
