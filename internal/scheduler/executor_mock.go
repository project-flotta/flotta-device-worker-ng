package scheduler

import (
	"context"

	"github.com/tupyy/device-worker-ng/internal/entity"
)

type MockExecutor struct {
	RunCount  int
	StopCount int
	futureCh  map[string]chan TaskState
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		futureCh: make(map[string]chan TaskState),
	}
}

func (e *MockExecutor) Run(ctx context.Context, w entity.Workload) *Future[TaskState] {
	e.RunCount++
	ch := make(chan TaskState)
	e.futureCh[w.ID()] = ch
	f := NewFuture(ch)
	ch <- TaskStateDeployed
	return f
}

func (e *MockExecutor) Stop(ctx context.Context, w entity.Workload) {
	e.StopCount++

	ch, ok := e.futureCh[w.ID()]
	if !ok {
		return
	}

	ch <- TaskStateStopped
}

func (e *MockExecutor) SendStateToTask(id string, state TaskState, resolveFuture bool) {
	ch, ok := e.futureCh[id]
	if !ok {
		return
	}
	ch <- state
	if state == TaskStateStopped || state == TaskStateExited || state == TaskStateUnknown {
		close(ch)
		delete(e.futureCh, id)
	}
}
