package godi_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/assurrussa/godi"
)

func TestLifecycleFailedStartThenShutdown(t *testing.T) {
	t.Parallel()
	l := godi.NewLifecycle()
	var calls []string
	startupErr := errors.New("startup failed")
	l.Append(godi.Hook{
		OnStart: func(context.Context) error { calls = append(calls, "start-a"); return nil },
		OnStop:  func(context.Context) error { calls = append(calls, "stop-a"); return nil },
	})
	l.Append(godi.Hook{
		OnStart: func(context.Context) error { calls = append(calls, "start-b"); return startupErr },
		OnStop:  func(context.Context) error { calls = append(calls, "stop-b"); return nil },
	})
	l.Append(godi.Hook{
		OnStart: func(context.Context) error { calls = append(calls, "start-c"); return nil },
		OnStop:  func(context.Context) error { calls = append(calls, "stop-c"); return nil },
	})
	for range 2 {
		if err := l.Start(context.Background()); !errors.Is(err, startupErr) {
			t.Fatalf("lost startup error: %v", err)
		}
		if err := l.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if want := []string{"start-a", "start-b", "stop-a"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("got %v, want %v", calls, want)
	}
}

func TestLifecycleRollbackContextAndErrors(t *testing.T) {
	t.Parallel()
	type contextKey struct{}
	const contextValue = "rollback-value"
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey{}, contextValue))
	defer cancel()
	l := godi.NewLifecycle()
	cleanupErr := errors.New("cleanup failed")
	stops := 0
	l.Append(godi.Hook{
		OnStart: func(context.Context) error { return nil },
		OnStop: func(ctx context.Context) error {
			stops++
			if ctx.Err() != nil || ctx.Value(contextKey{}) != contextValue {
				t.Errorf("rollback context is cancelled or lost values: %v", ctx.Err())
			}
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 30*time.Second {
				t.Error("rollback has no bounded cleanup deadline")
			}
			return cleanupErr
		},
	})
	l.Append(godi.Hook{OnStart: func(ctx context.Context) error { cancel(); return ctx.Err() }})
	err := l.Start(ctx)
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cleanupErr) {
		t.Fatalf("errors not joined: %v", err)
	}
	if err := l.Stop(context.Background()); !errors.Is(err, cleanupErr) {
		t.Fatalf("stop did not retain cleanup error: %v", err)
	}
	if stops != 1 {
		t.Fatalf("cleanup repeated %d times", stops)
	}
}

func TestLifecycleRepeatedCallsAndStopOnly(t *testing.T) {
	t.Parallel()
	for _, start := range []bool{false, true} {
		l := godi.NewLifecycle()
		starts, stops, stopOnly := 0, 0, 0
		l.Append(godi.Hook{
			OnStart: func(context.Context) error { starts++; return nil },
			OnStop:  func(context.Context) error { stops++; return nil },
		})
		l.Append(godi.Hook{OnStop: func(context.Context) error { stopOnly++; return nil }})
		if start {
			for range 2 {
				if err := l.Start(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
		}
		for range 2 {
			if err := l.Stop(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		want := 0
		if start {
			want = 1
		}
		if starts != want || stops != want || stopOnly != 1 {
			t.Fatalf("start=%v: starts=%d stops=%d stop-only=%d", start, starts, stops, stopOnly)
		}
		if err := l.Start(context.Background()); err == nil {
			t.Fatal("restarted stopped lifecycle")
		}
	}
}

func TestLifecycleFailureCleansStopOnlyHooks(t *testing.T) {
	t.Parallel()
	l := godi.NewLifecycle()
	stops := 0
	l.Append(godi.Hook{OnStart: func(context.Context) error { return errors.New("failure") }})
	l.Append(godi.Hook{OnStop: func(context.Context) error { stops++; return nil }})
	if err := l.Start(context.Background()); err == nil {
		t.Fatal("expected start error")
	}
	if err := l.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if stops != 1 {
		t.Fatalf("stop-only hook cleaned up %d times", stops)
	}
}

func TestLifecycleTransitionsRejectConcurrentAndReentrantCalls(t *testing.T) {
	t.Parallel()
	l := godi.NewLifecycle()
	entered := make(chan struct{})
	release := make(chan struct{})
	l.Append(godi.Hook{
		OnStart: func(ctx context.Context) error {
			if err := l.Stop(ctx); err == nil {
				return errors.New("reentrant stop accepted")
			}
			close(entered)
			<-release
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := l.Stop(ctx); err == nil {
				return errors.New("reentrant stop accepted")
			}
			return nil
		},
	})
	finished := make(chan error, 1)
	go func() { finished <- l.Start(context.Background()) }()
	select {
	case <-entered:
	case err := <-finished:
		t.Fatalf("start failed before synchronization: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("start deadlocked")
	}
	if err := l.Start(context.Background()); err == nil {
		t.Error("concurrent start accepted")
	}
	if err := l.Stop(context.Background()); err == nil {
		t.Error("concurrent stop accepted")
	}
	close(release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if err := l.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleAppendAfterStartPanics(t *testing.T) {
	t.Parallel()
	l := godi.NewLifecycle()
	if err := l.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Error("late hook registration accepted")
		}
	}()
	l.Append(godi.Hook{})
}
