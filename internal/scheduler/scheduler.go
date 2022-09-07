package scheduler

import (
	"context"
	"strings"
	"time"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/profile"
	"github.com/tupyy/device-worker-ng/internal/scheduler/common"
	"github.com/tupyy/device-worker-ng/internal/scheduler/reconcile"
	"go.uber.org/zap"
)

const (
	defaultHeartbeatPeriod = 1 * time.Second
	gracefullShutdown      = 5 * time.Second
)

type Scheduler struct {
	// jobs holds all the current jobs
	jobs *Store
	// executor
	executor common.Executor
	// runCancel is the cancel function of the run goroutine
	runCancel context.CancelFunc
	// reconciler
	reconciler common.Reconciler
	// profileEvaluationResults holds the latest profile evaluation results received from profile manager
	profileEvaluationResults []profile.ProfileEvaluationResult
	// futures holds the future for each reconciliation function in progress
	futures map[string]*entity.Future[entity.Result[entity.JobState]]
}

// New creates a new scheduler with the default heartbeat period of 2 seconds.
func New(executor common.Executor) *Scheduler {
	return newScheduler(executor, defaultHeartbeatPeriod)
}

// New creates a new scheduler with the hearbeat period provided by the user.
func NewWitHeartbeatPeriod(executor common.Executor, heartbeatPeriod time.Duration) *Scheduler {
	return newScheduler(executor, heartbeatPeriod)
}

func newScheduler(executor common.Executor, heartbeatPeriod time.Duration) *Scheduler {
	return &Scheduler{
		jobs:                     NewStore(),
		executor:                 executor,
		reconciler:               reconcile.New(),
		futures:                  make(map[string]*entity.Future[entity.Result[entity.JobState]]),
		profileEvaluationResults: make([]profile.ProfileEvaluationResult, 0),
	}
}

func (s *Scheduler) Start(ctx context.Context, input chan entity.Message, profileUpdateCh chan []profile.ProfileEvaluationResult) {
	runCtx, cancel := context.WithCancel(ctx)
	s.runCancel = cancel

	jobCh := make(chan entity.Option[[]entity.Workload])
	go func(ctx context.Context) {
		for {
			select {
			case message := <-input:
				switch message.Kind {
				case entity.WorkloadConfigurationMessage:
					val, ok := message.Payload.(entity.Option[[]entity.Workload])
					if !ok {
						zap.S().Errorf("mismatch message payload type. expected workload. got %v", message)
					}
					jobCh <- val
				}
			case <-ctx.Done():
				return
			}
		}
	}(runCtx)
	go s.run(runCtx, jobCh, profileUpdateCh)
}

func (s *Scheduler) Stop(ctx context.Context) {
	zap.S().Info("shutting down scheduler")

	// shutdown goroutines
	s.runCancel()

	zap.S().Info("scheduler shutdown")
}

func (s *Scheduler) run(ctx context.Context, input chan entity.Option[[]entity.Workload], profileCh chan []profile.ProfileEvaluationResult) {
	sync := make(chan struct{}, 1)

	heartbeat := time.NewTicker(defaultHeartbeatPeriod)

	for {
		select {
		case opt := <-input:
			if opt.None {
				for _, t := range s.jobs.ToList() {
					t.MarkForDeletion()
					t.SetTargetState(entity.ExitedState)
				}
				break
			}
			// add jobs
			jobsToRemove := substract(s.jobs.ToList(), opt.Value)
			newWorkloads := substract(opt.Value, s.jobs.ToList())
			for _, w := range newWorkloads {
				zap.S().Infow("new job", "job", w.String())
				j, err := s.createJob(w)
				if err != nil {
					zap.S().Errorw("failed to create job", "error", err)
					continue
				}
				s.jobs.Add(j)
				// evaluate job with the latest profile evaluation results
				if result, err := s.evaluate(j, s.profileEvaluationResults); err != nil {
					zap.S().Warnw("failed to evaluate profile", "error", err)
				} else if result {
					j.SetTargetState(entity.RunningState)
				}
			}
			// remove job which are not found in the EdgeWorkload manifest
			for _, j := range jobsToRemove {
				j.MarkForDeletion()
				j.SetTargetState(entity.ExitedState)
			}
		case <-sync:
			for _, j := range s.jobs.ToList() {
				// if there is already a reconciliation function in progress check if the future has been resolved.
				if future, found := s.futures[j.ID()]; found {
					/*
					* A resolved future means the reconciliation function returned.
					* The result could be either a new current state or an error in case that the executor failed to reconcile the job
					* In both cases, we remove the future. At the next heartbeat, a new reconciliation function will be executed with a new future.
					* If the future is not resolved, wait until the future is resolved.
					* */
					if result, isResolved := future.Poll(); isResolved {
						if result.Error != nil {
							zap.S().Errorw("failed to reconcile the job", "job_id", j.ID(), "error", result.Error)
						} else {
							j.SetCurrentState(result.Value)
						}
						delete(s.futures, j.ID())
					}

					continue
				}

				// from here on, we start a new reconciliation process for this job.
				// get the current state first
				state, err := s.executor.GetState(context.TODO(), j.Workload())
				if err != nil {
					zap.S().Errorw("failed to reconcile the job", "job_id", j.ID(), "error", err)
					continue
				}
				j.SetCurrentState(state)

				// If it is marked for deletion but it is still running then keep going with the reconciliation until we stop the job
				// and then remove it
				if j.IsMarkedForDeletion() && j.CurrentState().OneOf(entity.UnknownState, entity.ExitedState, entity.ReadyState) {
					zap.S().Infow("job removed", "job_id", j.ID())
					s.jobs.Delete(j)
					delete(s.futures, j.ID())
					continue
				}

				if !s.shouldReconcile(j) {
					continue
				}

				/* at this point we need to reconcile. There are couple of things to verify:
				* - if we need to restart the job, check if we can do that now or wait
				* - if the job needs to be executed, check if there is a cron attached to it and verify if we can started
				* Because cron is basically a retry at a certain time in future, a job cannot have *both* a cron and a retry attached.
				* */
				if j.ShouldRestart() && j.Retry() != nil {
					if !j.Retry().CanReconcile() {
						zap.S().DPanicw("job cannot be reconciled yet", "job_id", j.ID(), "next_retry", j.Retry().Next())
						continue
					}
					j.Retry().ComputeNext()
				}
				// look at the cron only if we need to run the job.
				if j.TargetState() == entity.RunningState && j.Cron() != nil {
					if !j.Cron().CanReconcile() {
						zap.S().Debugw("job cannot be reconciled yet", "job_id", j.ID(), "next_cron", j.Cron().Next())
						continue
					}
					j.Cron().ComputeNext()
				}
				// reconcile
				future := s.reconciler.Reconcile(context.Background(), j, s.executor)
				s.futures[j.ID()] = future
			}
		case results := <-profileCh:
			s.profileEvaluationResults = results
			zap.S().Infow("start evaluating job", "profile evaluation result", results)
			for _, j := range s.jobs.ToList() {
				// don't evaluate job marked for deletion
				if j.IsMarkedForDeletion() {
					continue
				}
				result, err := s.evaluate(j, results)
				if err != nil {
					zap.S().Warnw("failed to evaluate profiles", "job_id", j.ID(), "error", err)
					continue
				}
				switch result {
				case true:
					zap.S().Infow("job's profiles evaluated to true", "job_id", j.ID(), "job_profiles", j.Workload().Profiles(), "profile_evaluation_result", results)
					j.SetTargetState(entity.RunningState)
				case false:
					zap.S().Infow("job's profiles evaluated to false", "job_id", j.ID(), "job_profiles", j.Workload().Profiles(), "profile_evaluation_result", results)
					j.SetTargetState(entity.InactiveState)
				}
			}
		case <-heartbeat.C:
			sync <- struct{}{}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) createJob(w entity.Workload) (*entity.Job, error) {
	builder := entity.NewBuilder(w)
	if w.Cron() != "" {
		builder.WithCron(w.Cron())
	} else {
		builder.WithConstantRetry(20 * time.Second)
	}

	return builder.Build()
}

// shouldReconcile returns true if a job needs to be reconciled.
// The conditions for a job to be reconciled are:
//  - the job is idle, either stopped or never run, and the target_state is RunningState
//  - the job is running and needs to be stopped (i.e target_state one of UnknownState, InactiveState, ExitedState)
func (s *Scheduler) shouldReconcile(j *entity.Job) bool {
	if j.TargetState() == entity.RunningState && j.CurrentState().OneOf(entity.ReadyState, entity.InactiveState, entity.ExitedState, entity.UnknownState) {
		return true
	}

	if j.TargetState().OneOf(entity.ExitedState, entity.InactiveState) && j.CurrentState() == entity.RunningState {
		return true
	}

	return false
}

func (s *Scheduler) evaluate(j *entity.Job, results []profile.ProfileEvaluationResult) (bool, error) {
	if len(j.Workload().Profiles()) == 0 || len(results) == 0 {
		return true, nil
	}

	// make a map with job profile conditions
	m := make(map[string]string)
	for _, p := range j.Workload().Profiles() {
		conditions := strings.Join(p.Conditions, ",")
		m[p.Name] = conditions
	}

	// for each profile's condition evaluated to true try to find it in the job conditions
	sum := 0
	for _, result := range results {
		jobProfile, found := m[result.Name]
		if !found {
			continue
		}

		for _, condition := range result.ConditionsResults {
			if condition.Error != nil {
				return false, condition.Error
			}
			if condition.Value && strings.Contains(jobProfile, condition.Name) && condition.Error == nil {
				sum++
				break
			}
		}
	}

	// if at least one condition for each job's profile is true the sum
	// must be equal to number of profiles
	// in this case we consider that the job passed the evaluation
	return sum == len(j.Workload().Profiles()), nil
}
