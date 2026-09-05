package godi

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

type Hook struct {
	OnStart func(context.Context) error
	OnStop  func(context.Context) error
}

// HookPanicError reports a recovered panic in an OnStart or OnStop callback.
// Index is the zero-based registration index. Stack is captured at the panic.
type HookPanicError struct {
	Phase string
	Index int
	Value any
	Stack []byte
}

func (e *HookPanicError) Error() string {
	return fmt.Sprintf("lifecycle %s hook %d panicked: %v\n%s", e.Phase, e.Index, e.Value, e.Stack)
}

// Unwrap preserves errors.Is/errors.As when the panic value is an error.
func (e *HookPanicError) Unwrap() error {
	err, _ := e.Value.(error)
	return err
}

type indexedHook struct {
	Hook
	index int
}

type lifecycleState uint8

const (
	lifecycleIdle lifecycleState = iota
	lifecycleStarting
	lifecycleRunning
	lifecycleStopping
	lifecycleStopped
)

// Lifecycle manages a single start/stop cycle. It must not be copied after use.
// Concurrent or reentrant Start/Stop calls during a transition return an error.
// Hooks run without the lifecycle mutex held.
type Lifecycle struct {
	mu       sync.Mutex
	hooks    []Hook
	active   []indexedHook
	state    lifecycleState
	startErr error
	stopErr  error
}

func NewLifecycle() *Lifecycle {
	return &Lifecycle{}
}

// Append registers a hook before the first Start or Stop. It panics after that
// boundary, because a late hook could otherwise miss activation or cleanup.
func (l *Lifecycle) Append(h Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.state != lifecycleIdle {
		panic("cannot append hooks after lifecycle has started or stopped")
	}
	l.hooks = append(l.hooks, h)
}

// Start activates hooks in registration order. Repeating a successful Start is
// a no-op; a failed Start returns its original error on subsequent calls.
func (l *Lifecycle) Start(ctx context.Context) error {
	l.mu.Lock()
	if l.state != lifecycleIdle {
		err := l.startStateError()
		l.mu.Unlock()
		return err
	}
	l.state = lifecycleStarting
	hooks := append([]Hook(nil), l.hooks...)
	l.mu.Unlock()

	active := make([]bool, len(hooks))
	for i, hook := range hooks {
		// Stop-only hooks represent resources acquired outside Start.
		active[i] = hook.OnStart == nil
	}
	for i, hook := range hooks {
		if hook.OnStart == nil {
			continue
		}
		err := ctx.Err()
		if err == nil {
			err = runHook(ctx, hook.OnStart, "OnStart", i)
		}
		if err != nil {
			return l.rollback(ctx, activeHooks(hooks, active), err)
		}
		active[i] = true
	}

	l.mu.Lock()
	l.active = activeHooks(hooks, active)
	l.state = lifecycleRunning
	l.mu.Unlock()
	return nil
}

// startStateError is called with mu held.
func (l *Lifecycle) startStateError() error {
	switch l.state {
	case lifecycleRunning:
		return nil
	case lifecycleStopped:
		if l.startErr != nil {
			return l.startErr
		}
		return errors.New("cannot restart a stopped lifecycle")
	default:
		return errors.New("lifecycle transition already in progress")
	}
}

func (l *Lifecycle) rollback(ctx context.Context, hooks []indexedHook, startErr error) error {
	// Preserve context values, but allow cleanup after startup cancellation.
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	stopErr := l.stopStarted(cleanupCtx, hooks)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.startErr = errors.Join(startErr, stopErr)
	l.stopErr = stopErr
	l.state = lifecycleStopped
	return l.startErr
}

// Stop cleans up active hooks in reverse registration order, at most once.
// Before Start, only stop-only hooks run. Repeated calls return the saved error.
func (l *Lifecycle) Stop(ctx context.Context) error {
	l.mu.Lock()
	switch l.state {
	case lifecycleStopped:
		err := l.stopErr
		l.mu.Unlock()
		return err
	case lifecycleStarting, lifecycleStopping:
		l.mu.Unlock()
		return errors.New("lifecycle transition already in progress")
	case lifecycleIdle:
		for i, hook := range l.hooks {
			if hook.OnStart == nil {
				l.active = append(l.active, indexedHook{Hook: hook, index: i})
			}
		}
	case lifecycleRunning:
		// The active hooks were recorded by Start.
	}
	l.state = lifecycleStopping
	hooks := l.active
	l.active = nil
	l.mu.Unlock()

	err := l.stopStarted(ctx, hooks)
	l.mu.Lock()
	l.stopErr = err
	l.state = lifecycleStopped
	l.mu.Unlock()
	return err
}

func activeHooks(hooks []Hook, active []bool) []indexedHook {
	result := make([]indexedHook, 0, len(hooks))
	for i, hook := range hooks {
		if active[i] {
			result = append(result, indexedHook{Hook: hook, index: i})
		}
	}
	return result
}

func (l *Lifecycle) stopStarted(ctx context.Context, hooks []indexedHook) error {
	var stopErr error
	for i := len(hooks) - 1; i >= 0; i-- {
		hook := hooks[i]
		if hook.OnStop == nil {
			continue
		}
		if err := runHook(ctx, hook.OnStop, "OnStop", hook.index); err != nil {
			stopErr = errors.Join(stopErr, err)
		}
	}
	return stopErr
}

// Catch each callback separately so a cleanup panic cannot skip other hooks.
func runHook(ctx context.Context, callback func(context.Context) error, phase string, index int) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = &HookPanicError{Phase: phase, Index: index, Value: value, Stack: debug.Stack()}
		}
	}()
	return callback(ctx)
}
