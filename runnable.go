package godi

import (
	"context"

	"go.uber.org/dig"
)

type Runnable struct {
	OnStart func(context.Context) error
	OnStop  func(context.Context) error
}

const runnableGroup = "goshared_di_runnable"

type runnables struct {
	dig.In
	Runnables []Runnable `group:"goshared_di_runnable"`
}
