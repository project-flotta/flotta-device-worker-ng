package scheduler

import (
	"errors"
	"sync"
)

type Element interface {
	Name() string
	ID() string
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

func (s *Store[T]) FindByID(id string) (T, bool) {
	for i, t := range s.tasks {
		if t.ID() == id {
			return s.tasks[i], true
		}
	}
	var none T
	return none, false
}

func (s *Store[T]) FindByName(name string) (T, bool) {
	for i, t := range s.tasks {
		if t.Name() == name {
			return s.tasks[i], true
		}
	}
	var none T
	return none, false
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

func (s *Store[T]) ToList() []T {
	return s.tasks[:]
}

func (s *Store[T]) Clone() *Store[T] {
	return &Store[T]{
		tasks: s.tasks[:],
	}
}

func (s *Store[T]) index(id string) (int, error) {
	for i, t := range s.tasks {
		if t.ID() == id {
			return i, nil
		}
	}

	return 0, errors.New("element not found")
}
