package scheduler

import (
	"context"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/task"
)

type MockExecutor struct {
	RunCount  int
	StopCount int
	futureCh  map[string]chan task.State
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		futureCh: make(map[string]chan task.State),
	}
}

func (e *MockExecutor) Run(ctx context.Context, w entity.Workload) *Future[task.State] {
	e.RunCount++
	ch := make(chan task.State)
	e.futureCh[w.ID()] = ch
	f := NewFuture(ch)
	ch <- task.DeployedState
	return f
}

func (e *MockExecutor) Stop(ctx context.Context, w entity.Workload) {
	e.StopCount++

	ch, ok := e.futureCh[w.ID()]
	if !ok {
		return
	}

	ch <- task.StoppedState
	close(ch)
	delete(e.futureCh, w.ID())
}

func (e *MockExecutor) SendStateToTask(id string, state task.State, resolveFuture bool) {
	ch, ok := e.futureCh[id]
	if !ok {
		return
	}
	ch <- state
	if state.OneOf(task.ExitedState, task.StoppedState) {
		close(ch)
		delete(e.futureCh, id)
	}
}
