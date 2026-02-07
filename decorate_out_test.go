package godi_test

import (
	"testing"

	"go.uber.org/dig"

	"github.com/assurrussa/godi"
)

func TestDecorateWithDigOutNamedSlot(t *testing.T) {
	t.Parallel()

	type out struct {
		dig.Out
		S string `name:"n"`
	}

	cnt, err := godi.NewContainer(godi.WithDependencies(
		godi.CollectDependencies(
			godi.NewDependency(func() string { return "base" }, godi.WithName("n")),
			godi.Decorate(func(in struct {
				dig.In
				S string `name:"n"`
			},
			) out {
				return out{S: in.S + "!"}
			}),
		),
	))
	if err != nil {
		t.Fatalf("NewContainer error: %v", err)
	}

	var got string
	if err := cnt.Invoke(func(in struct {
		dig.In
		S string `name:"n"`
	},
	) {
		got = in.S
	}); err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if got != "base!" {
		t.Fatalf("expected decorated named value, got %q", got)
	}
}

func TestDecorateWithDigOutGroupOutputIsRejected(t *testing.T) {
	t.Parallel()

	type out struct {
		dig.Out
		Items []string `group:"items"`
	}

	_, err := godi.NewContainer(godi.WithDependencies(
		godi.CollectDependencies(
			godi.NewDependency(func() []string { return []string{"x"} }, godi.WithGroup("items")),
			godi.Decorate(func(in struct {
				dig.In
				Items []string `group:"items"`
			},
			) out {
				return out{Items: in.Items}
			}),
		),
	))
	if err == nil {
		t.Fatal("expected error for decorator with group outputs")
	}
}
