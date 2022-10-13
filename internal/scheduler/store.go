package scheduler

import (
	"errors"
	"sync"

	entity "github.com/tupyy/device-worker-ng/internal/entity"
)

// it is *not* thread safe
type Store struct {
	lock sync.Mutex
	jobs []*entity.Job
}

func NewStore() *Store {
	return &Store{
		jobs: make([]*entity.Job, 0, 3),
	}
}

func (s *Store) Len() int {
	s.lock.Lock()
	defer s.lock.Unlock()

	return len(s.jobs)
}

func (s *Store) Get(idx int) (*entity.Job, bool) {
	if idx >= len(s.jobs) {
		return nil, false
	}
	return s.jobs[idx], true
}

func (s *Store) Find(id string) (*entity.Job, bool) {
	for i, j := range s.jobs {
		if j.ID() == id {
			return s.jobs[i], true
		}
	}
	var none *entity.Job
	return none, false
}

func (s *Store) Delete(element *entity.Job) *entity.Job {
	s.lock.Lock()
	defer s.lock.Unlock()

	var none *entity.Job
	idx, err := s.index(element.ID())
	if err != nil {
		return none
	}

	task := s.jobs[idx]
	s.jobs = append(s.jobs[:idx], s.jobs[idx+1:]...)
	return task
}

func (s *Store) Add(t *entity.Job) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.jobs = append(s.jobs, t)
}

func (s *Store) ToList() []*entity.Job {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.jobs[:]
}

func (s *Store) index(id string) (int, error) {
	for i, t := range s.jobs {
		if t.ID() == id {
			return i, nil
		}
	}

	return 0, errors.New("element not found")
}
