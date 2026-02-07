package godi

import (
	"context"
	"errors"
	"sync"
)

type Hook struct {
	OnStart func(context.Context) error
	OnStop  func(context.Context) error
}

// Lifecycle manages start/stop hooks in order (start) and reverse order (stop).
type Lifecycle struct {
	mu    sync.Mutex
	hooks []Hook
}

func NewLifecycle() *Lifecycle {
	return &Lifecycle{}
}

func (l *Lifecycle) Append(h Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = append(l.hooks, h)
}

func (l *Lifecycle) Start(ctx context.Context) error {
	hooks := l.snapshot()
	for i, hook := range hooks {
		if hook.OnStart == nil {
			continue
		}
		if err := hook.OnStart(ctx); err != nil {
			_ = l.stopStarted(ctx, hooks[:i])
			return err
		}
	}
	return nil
}

func (l *Lifecycle) Stop(ctx context.Context) error {
	hooks := l.snapshot()
	return l.stopStarted(ctx, hooks)
}

func (l *Lifecycle) snapshot() []Hook {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Hook(nil), l.hooks...)
}

func (l *Lifecycle) stopStarted(ctx context.Context, hooks []Hook) error {
	var stopErr error
	for i := len(hooks) - 1; i >= 0; i-- {
		hook := hooks[i]
		if hook.OnStop == nil {
			continue
		}
		if err := hook.OnStop(ctx); err != nil {
			stopErr = errors.Join(stopErr, err)
		}
	}
	return stopErr
}
