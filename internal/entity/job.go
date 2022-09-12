package entity

import (
	"encoding/json"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/robfig/cron"
	"go.uber.org/zap"
)

type CronJob struct {
	// next time the job can be reconciled
	next     time.Time
	schedule cron.Schedule
}

func (cj *CronJob) ComputeNext() {
	cj.next = cj.schedule.Next(time.Now())
}

func (cj *CronJob) CanReconcile() bool {
	return time.Now().After(cj.next)
}

func (cj *CronJob) Next() time.Time {
	return cj.next
}

type RetryJob struct {
	MarkedForRestart bool
	next             time.Time
	b                backoff.BackOff
}

func (rj *RetryJob) ComputeNext() {
	rj.next = time.Now().Add(rj.b.NextBackOff())
}

func (rj *RetryJob) CanReconcile() bool {
	return time.Now().After(rj.next)
}

func (rj *RetryJob) Next() time.Time {
	return rj.next
}

type Job struct {
	// workload
	workload Workload
	// currentState holds the current state of the job
	currentState JobState
	// targetState holds the desired next state of the job
	// nextState is mutated by the scheduler when it wants to run/stop the workload
	targetState JobState
	// markedForDeletion is true if the job has to be deleted
	markedForDeletion bool
	cron              *CronJob
	retry             *RetryJob
}

func (j *Job) SetTargetState(state JobState) error {
	j.targetState = state
	return nil
}

func (j *Job) TargetState() JobState {
	return j.targetState
}

func (j *Job) CurrentState() JobState {
	return j.currentState
}

func (j *Job) SetCurrentState(currentState JobState) {
	// if we allow to set the unknow state when the job is ready the restart will be activated
	if currentState == UnknownState && j.currentState == ReadyState {
		return
	}

	j.currentState = currentState

	if j.ShouldRestart() && j.Retry() != nil {
		if !j.Retry().MarkedForRestart {
			j.Retry().ComputeNext()
			j.Retry().MarkedForRestart = true
			zap.S().Debugw("marked job for restart", "job_id", j.ID(), "next_restart_after", j.Retry().Next())
		}
	}

	if currentState == RunningState && j.Retry() != nil {
		j.Retry().MarkedForRestart = false
	}
}

func (j *Job) ShouldRestart() bool {
	return j.CurrentState().OneOf(ExitedState, UnknownState) && j.TargetState() == RunningState
}

func (j *Job) String() string {
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

func (j *Job) ID() string {
	return j.workload.ID()
}

func (j *Job) Workload() Workload {
	return j.workload
}

func (j *Job) MarkForDeletion() {
	zap.S().Debugw("job marked for deletion", "job_id", j.ID())
	j.markedForDeletion = true
}

func (j *Job) IsMarkedForDeletion() bool {
	return j.markedForDeletion
}

func (j *Job) Cron() *CronJob {
	return j.cron
}

func (j *Job) Retry() *RetryJob {
	return j.retry
}

type Builder struct {
	w            Workload
	cronSpec     string
	retryBackoff backoff.BackOff
}

func NewBuilder(w Workload) *Builder {
	return &Builder{w: w}
}

func (jb *Builder) WithExponentialRetry(initialInterval time.Duration, maxInterval time.Duration, multiplier float64) *Builder {
	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = initialInterval
	exp.MaxInterval = maxInterval
	exp.Multiplier = multiplier
	jb.retryBackoff = exp
	return jb
}

func (jb *Builder) WithConstantRetry(retryInterval time.Duration) *Builder {
	cst := backoff.NewConstantBackOff(retryInterval)
	jb.retryBackoff = cst
	return jb
}

func (jb *Builder) WithCron(cronSpec string) *Builder {
	jb.cronSpec = cronSpec
	return jb
}

func (jb *Builder) Build() (*Job, error) {
	j := &Job{
		workload:     jb.w,
		currentState: ReadyState,
		targetState:  ReadyState,
	}

	if jb.cronSpec != "" {
		specParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		sched, err := specParser.Parse(jb.cronSpec)
		if err != nil {
			return nil, err
		}
		j.cron = &CronJob{
			schedule: sched,
			next:     sched.Next(time.Now()),
		}
	}

	if jb.retryBackoff != nil {
		j.retry = &RetryJob{
			next: time.Now(),
			b:    jb.retryBackoff,
		}
	}

	return j, nil
}
