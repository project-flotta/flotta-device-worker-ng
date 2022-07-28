package entity

import (
	"github.com/google/uuid"
)

type TaskStatus int

const (
	ReadyStatus TaskStatus = iota
	DeployingStatus
	RunningStatus
	StoppingStatus
	StoppedStatus
	ExitedStatus
)

// Task is a wrapper around the actual Workload
// providing extra fields to be used by the scheduler.
type Task struct {
	Name          string
	ID            string
	Workload      Workload
	CurrentStatus TaskStatus
	NextStatus    TaskStatus
}

func NewTask(name string, w Workload) Task {
	id := uuid.New().String()

	return Task{
		Name:          name,
		ID:            id,
		Workload:      w,
		CurrentStatus: ReadyStatus,
		NextStatus:    ReadyStatus,
	}
}

type ExecutionResult struct {
	TaskID string
	Status TaskStatus
	Error  error
}
