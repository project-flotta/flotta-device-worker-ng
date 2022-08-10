package containers

import (
	"errors"
	"sync"
)

type Element interface {
	ID() string
}

// A naive implementation of the iterator.
// it is *not* thread safe.
// Calling Next from different goroutines, could lead to a wierd behaviour.
type Iter[T Element] struct {
	lock sync.Mutex
	idx  int
	s    *Store[T]
}

func (i *Iter[T]) Next() (T, bool) {
	i.lock.Lock()
	defer i.lock.Unlock()

	var none T
	if i.idx >= i.s.Len() {
		return none, false
	}

	oldIdx := i.idx
	i.idx++

	return i.s.Get(oldIdx)
}

func (i *Iter[T]) HasNext() bool {
	return i.idx < i.s.Len()
}

// it is *not* thread safe
type Store[T Element] struct {
	lock  sync.Mutex
	tasks []T
}

func NewStore[T Element]() *Store[T] {
	return &Store[T]{
		tasks: make([]T, 0, 3),
	}
}

func (s *Store[T]) Iter() *Iter[T] {
	return &Iter[T]{
		idx: 0,
		s:   s.clone(),
	}
}

func (s *Store[T]) Len() int {
	return len(s.tasks)
}

func (s *Store[T]) Get(idx int) (T, bool) {
	var none T
	if idx >= len(s.tasks) {
		return none, false
	}
	return s.tasks[idx], true
}

func (s *Store[T]) Find(name string) (T, bool) {
	var none T
	idx, err := s.index(name)
	if err != nil {
		return none, false
	}
	return s.tasks[idx], true
}

func (s *Store[T]) Delete(element T) T {
	var none T
	idx, err := s.index(element.ID())
	if err != nil {
		return none
	}

	task := s.tasks[idx]
	s.tasks = append(s.tasks[:idx], s.tasks[idx+1:]...)
	return task
}

func (s *Store[T]) Add(t T) {
	s.tasks = append(s.tasks, t)
}

func (s *Store[T]) clone() *Store[T] {
	return &Store[T]{
		tasks: s.tasks[:],
	}
}

func (s *Store[T]) index(name string) (int, error) {
	for i, t := range s.tasks {
		if t.ID() == name {
			return i, nil
		}
	}

	return 0, errors.New("element not found")
}
