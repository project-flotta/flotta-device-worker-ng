package scheduler

import (
	"fmt"

	"github.com/tupyy/device-worker-ng/internal/scheduler/containers"
)

type Result[T any] struct {
	Value T
	ready bool
}

func (r Result[T]) IsReady() bool {
	return r.ready
}

func (r Result[T]) IsPending() bool {
	return !r.ready
}

type Future[T any] struct {
	input       chan T
	inputClosed bool
	values      containers.Queue[T]
}

func (f *Future[T]) Resolved() bool {
	return f.inputClosed && f.values.Size() == 0
}

func NewFuture[T any](input chan T) *Future[T] {
	f := &Future[T]{
		input:       input,
		inputClosed: false,
	}

	go func() {
		for value := range f.input {
			f.values.Push(value)
		}
		f.inputClosed = true
	}()

	return f
}

func (f *Future[T]) Poll() (Result[T], error) {
	if f.Resolved() {
		return Result[T]{}, fmt.Errorf("future already resolved")
	}

	if f.values.Size() > 0 {
		return Result[T]{
			Value: f.values.Pop(),
			ready: true,
		}, nil
	}

	return Result[T]{
		ready: false,
	}, nil
}
