package godi_test

import (
	"context"
	"errors"
	"testing"

	"github.com/assurrussa/godi"
)

func TestLifecycleStartStopOrder(t *testing.T) {
	t.Parallel()

	l := godi.NewLifecycle()
	var calls []string

	l.Append(godi.Hook{
		OnStart: func(context.Context) error { calls = append(calls, "s1"); return nil },
		OnStop:  func(context.Context) error { calls = append(calls, "t1"); return nil },
	})
	l.Append(godi.Hook{
		OnStart: func(context.Context) error { calls = append(calls, "s2"); return nil },
		OnStop:  func(context.Context) error { calls = append(calls, "t2"); return nil },
	})

	if err := l.Start(context.Background()); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if err := l.Stop(context.Background()); err != nil {
		t.Fatalf("Stop error: %v", err)
	}

	want := []string{"s1", "s2", "t2", "t1"}
	if len(calls) != len(want) {
		t.Fatalf("expected %v, got %v", want, calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, calls)
		}
	}
}

func TestLifecycleStartFailureStopsStartedHooks(t *testing.T) {
	t.Parallel()

	l := godi.NewLifecycle()
	var calls []string

	l.Append(godi.Hook{
		OnStart: func(context.Context) error { calls = append(calls, "s1"); return nil },
		OnStop:  func(context.Context) error { calls = append(calls, "t1"); return nil },
	})
	l.Append(godi.Hook{
		OnStart: func(context.Context) error { calls = append(calls, "s2"); return errors.New("boom") },
		OnStop:  func(context.Context) error { calls = append(calls, "t2"); return nil },
	})

	if err := l.Start(context.Background()); err == nil {
		t.Fatal("expected Start to fail")
	}

	// Only the successfully started hooks are stopped.
	want := []string{"s1", "s2", "t1"}
	if len(calls) != len(want) {
		t.Fatalf("expected %v, got %v", want, calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, calls)
		}
	}
}

func TestLifecycleStopJoinsErrors(t *testing.T) {
	t.Parallel()

	l := godi.NewLifecycle()
	e1 := errors.New("e1")
	e2 := errors.New("e2")
	l.Append(godi.Hook{OnStop: func(context.Context) error { return e1 }})
	l.Append(godi.Hook{OnStop: func(context.Context) error { return e2 }})

	err := l.Stop(context.Background())
	if err == nil {
		t.Fatal("expected stop error")
	}
	if !errors.Is(err, e1) {
		t.Fatalf("expected joined error to contain e1, got %v", err)
	}
	if !errors.Is(err, e2) {
		t.Fatalf("expected joined error to contain e2, got %v", err)
	}
}
