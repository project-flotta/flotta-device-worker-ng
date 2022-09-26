package entity

import (
	"encoding/json"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/robfig/cron"
	"go.uber.org/zap"
)

// CpuResource holds the quota and the period.
// Quota - CPU hardcap limit (in usecs). Allowed cpu time in a given period.
// Period - CPU period to be used for hardcapping (in usecs).
type CpuResource Tuple[uint64, uint64]

func (c CpuResource) Equal(other CpuResource) bool {
	return c.Value1 == other.Value1 && c.Value2 == other.Value2
}

type HookType int

const (
	PreSetCurrentState HookType = iota
	PostSetCurrentState
	PreSetTargetState
	PostSetTargetState
)

// JobHook is a hook run when either target or current state is changed
type JobStateHook func(j *Job, s JobState)

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
	// targetState is mutated by the scheduler when it wants to run/stop the workload
	targetState JobState
	// markedForDeletion is true if the job has to be deleted
	markedForDeletion bool
	cron              *CronJob
	retry             *RetryJob
	hooks             []Pair[HookType, JobStateHook]
	// current resources
	currentResources CpuResource
	// target resources
	targetResources CpuResource
}

func (j *Job) SetTargetState(state JobState) error {
	runHooksFn := func(hookType HookType) {
		for _, hook := range j.hooks {
			if hook.Name == hookType {
				hook.Value(j, state)
			}
		}
	}
	runHooksFn(PreSetTargetState)
	j.targetState = state
	runHooksFn(PostSetTargetState)
	return nil
}

func (j *Job) TargetState() JobState {
	return j.targetState
}

func (j *Job) CurrentState() JobState {
	return j.currentState
}

func (j *Job) SetCurrentState(state JobState) {
	runHooksFn := func(hookType HookType) {
		for _, hook := range j.hooks {
			if hook.Name == hookType {
				hook.Value(j, state)
			}
		}
	}

	runHooksFn(PreSetCurrentState)

	// if we allow to set the unknow state when the job is ready the restart will be activated
	if state == UnknownState && (j.currentState == ReadyState || j.currentState == InactiveState) {
		return
	}

	// this is almost a hack until I found something better.
	// The idea is not to set the current state to unknown if the target state is inactive.
	// When the target state is inactive it means we stopped the job ourselves but the sync branch of the scheduler will report unknown state
	// because the job was removed from podman/k8s therefore if we allow the current state to be unknown when the restart will happen, the reply backoff will be looked at
	// which is wrong. We should be able to restart the job without hitting the backoff function.
	if state.OneOf(UnknownState, ExitedState) && j.TargetState() == InactiveState {
		j.currentState = InactiveState
		return
	}

	j.currentState = state

	runHooksFn(PostSetCurrentState)
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

func (j *Job) SetCurrentResources(r CpuResource) {
	j.currentResources = r
}

func (j *Job) CurrentResources() CpuResource {
	return j.currentResources
}

func (j *Job) SetTargetResources(r CpuResource) {
	zap.S().Debugw("set target resources", "job_id", j.ID(), "target_resources", r)
	j.targetResources = r
}

func (j *Job) TargetResources() CpuResource {
	return j.targetResources
}

func (j *Job) Clone() *Job {
	clone := &Job{
		workload:          j.workload,
		currentState:      j.currentState,
		targetState:       j.targetState,
		markedForDeletion: j.markedForDeletion,
		hooks:             j.hooks[:],
		currentResources:  j.currentResources,
		targetResources:   j.targetResources,
	}

	if j.cron != nil {
		clone.cron = &CronJob{
			next:     time.UnixMicro(j.cron.next.UnixMicro()),
			schedule: j.cron.schedule,
		}
	}

	if j.retry != nil {
		clone.retry = &RetryJob{
			next:             time.UnixMicro(j.retry.next.UnixMicro()),
			MarkedForRestart: j.retry.MarkedForRestart,
			b:                j.retry.b,
		}
	}

	return clone
}

type Builder struct {
	w            Workload
	cronSpec     string
	retryBackoff backoff.BackOff
	hooks        []Pair[HookType, JobStateHook]
}

func NewBuilder(w Workload) *Builder {
	return &Builder{w: w, hooks: make([]Pair[HookType, JobStateHook], 0)}
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

func (jb *Builder) AddHook(hookType HookType, fn JobStateHook) *Builder {
	jb.hooks = append(jb.hooks, Pair[HookType, JobStateHook]{
		Name:  hookType,
		Value: fn,
	})
	return jb
}

func (jb *Builder) Build() (*Job, error) {
	j := &Job{
		workload:     jb.w,
		currentState: ReadyState,
		targetState:  ReadyState,
		hooks:        make([]Pair[HookType, JobStateHook], 0),
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

	if len(jb.hooks) > 0 {
		j.hooks = append(j.hooks, jb.hooks...)
	}

	return j, nil
}
