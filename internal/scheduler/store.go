package scheduler

import (
	"errors"
	"sync"

	"github.com/tupyy/device-worker-ng/internal/scheduler/job"
)

// it is *not* thread safe
type Store struct {
	lock sync.Mutex
	jobs []*job.DefaultJob
}

func NewStore() *Store {
	return &Store{
		jobs: make([]*job.DefaultJob, 0, 3),
	}
}

func (s *Store) Len() int {
	return len(s.jobs)
}

func (s *Store) Get(idx int) (*job.DefaultJob, bool) {
	if idx >= len(s.jobs) {
		return nil, false
	}
	return s.jobs[idx], true
}

func (s *Store) Find(id string) (*job.DefaultJob, bool) {
	for i, j := range s.jobs {
		if j.ID() == id {
			return s.jobs[i], true
		}
	}
	var none *job.DefaultJob
	return none, false
}

func (s *Store) Delete(element *job.DefaultJob) *job.DefaultJob {
	var none *job.DefaultJob
	idx, err := s.index(element.ID())
	if err != nil {
		return none
	}

	task := s.jobs[idx]
	s.jobs = append(s.jobs[:idx], s.jobs[idx+1:]...)
	return task
}

func (s *Store) Add(t *job.DefaultJob) {
	s.jobs = append(s.jobs, t)
}

func (s *Store) ToList() []*job.DefaultJob {
	return s.jobs[:]
}

func (s *Store) Clone() *Store {
	return &Store{
		jobs: s.jobs[:],
	}
}

func (s *Store) index(id string) (int, error) {
	for i, t := range s.jobs {
		if t.ID() == id {
			return i, nil
		}
	}

	return 0, errors.New("element not found")
}
