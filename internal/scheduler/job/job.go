package job

import (
	"encoding/json"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/robfig/cron"
	"github.com/tupyy/device-worker-ng/internal/entity"
	"go.uber.org/zap"
)

type CronJob struct {
	// next time the job can be reconciled
	Next     time.Time
	schedule cron.Schedule
}

func (cj *CronJob) ComputeNext() {
	cj.Next = cj.schedule.Next(time.Now())
}

type RetryJob struct {
	NextRetry time.Time
	b         backoff.BackOff
}

func (rj *RetryJob) ComputeNextRetryTime() {
	rj.NextRetry = rj.NextRetry.Add(rj.b.NextBackOff())
}

type DefaultJob struct {
	// workload
	workload entity.Workload
	// currentState holds the current state of the job
	currentState State
	// targetState holds the desired next state of the job
	// nextState is mutated by the scheduler when it wants to run/stop the workload
	targetState State
	// markedForDeletion is true if the job has to be deleted
	markedForDeletion bool
	cron              *CronJob
	retry             *RetryJob
}

func (j *DefaultJob) SetTargetState(state State) error {
	zap.S().Debugw("set target state", "job_id", j.ID(), "target_state", state)
	j.targetState = state
	return nil
}

func (j *DefaultJob) TargetState() State {
	return j.targetState
}

func (j *DefaultJob) CurrentState() State {
	return j.currentState
}

func (j *DefaultJob) SetCurrentState(currentState State) {
	j.currentState = currentState
}

func (j *DefaultJob) String() string {
	job := struct {
		ID           string `json:"id"`
		Workload     string `json:"workload"`
		CurrentState string `json:"current_state"`
		TargetState  string `json:"target_state"`
	}{
		ID:           j.ID(),
		Workload:     j.workload.String(),
		CurrentState: j.CurrentState().String(),
		TargetState:  j.TargetState().String(),
	}

	json, err := json.Marshal(job)
	if err != nil {
		return "error marshaling"
	}

	return string(json)
}

func (j *DefaultJob) ID() string {
	return j.workload.ID()
}

func (j *DefaultJob) Workload() entity.Workload {
	return j.workload
}

func (j *DefaultJob) MarkForDeletion() {
	j.markedForDeletion = true
}

func (j *DefaultJob) IsMarkedForDeletion() bool {
	return j.markedForDeletion
}

func (j *DefaultJob) Cron() *CronJob {
	return j.cron
}

func (j *DefaultJob) Retry() *RetryJob {
	return j.retry
}
