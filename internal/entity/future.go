package entity

import (
	"context"
	"sync"
)

type Future[T any] struct {
	input       chan T
	inputClosed bool
	value       T
	CancelFunc  context.CancelFunc
	lock        sync.Mutex
}

func (f *Future[T]) Resolved() bool {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.inputClosed
}

func NewFuture[T any](input chan T) *Future[T] {
	f := &Future[T]{
		input:       input,
		inputClosed: false,
	}

	go func() {
		for value := range f.input {
			f.value = value
		}
		f.lock.Lock()
		f.inputClosed = true
		f.lock.Unlock()
	}()

	return f
}

func (f *Future[T]) Poll() (value T, isResolved bool) {
	if f.Resolved() {
		return f.value, true
	}

	var none T
	return none, false
}
