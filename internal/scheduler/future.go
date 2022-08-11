package scheduler

import (
	"fmt"
	"sync"
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
	input    chan T
	done     bool
	hasValue bool
	value    T
	lock     sync.Mutex
}

func (f *Future[T]) Resolved() bool {
	return f.done
}

func NewFuture[T any](input chan T) *Future[T] {
	f := &Future[T]{
		input:    input,
		hasValue: false,
		done:     false,
	}

	go func() {
		for {
			value, more := <-f.input
			f.done = !more
			if !more {
				return
			}
			f.set(value, true)
		}
	}()

	return f
}

func (f *Future[T]) Poll() (Result[T], error) {
	if f.done && !f.hasValue {
		return Result[T]{}, fmt.Errorf("future already resolved")
	}

	if f.hasValue {
		return Result[T]{
			Value: f.consume(),
			ready: true,
		}, nil
	}

	return Result[T]{
		ready: false,
	}, nil
}

func (f *Future[T]) set(v T, hasValue bool) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.hasValue = hasValue
	f.value = v
}

func (f *Future[T]) consume() T {
	val := f.value
	var none T
	f.set(none, false)
	return val
}
