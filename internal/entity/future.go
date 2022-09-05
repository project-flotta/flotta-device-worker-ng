package entity

type Future[T any] struct {
	input       chan T
	inputClosed bool
	value       T
}

func (f *Future[T]) Resolved() bool {
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
		f.inputClosed = true
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
