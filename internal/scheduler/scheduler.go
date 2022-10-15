package scheduler

import (
	"context"
	"errors"
	"fmt"
	reflect "reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/profile"
	"github.com/tupyy/device-worker-ng/internal/resources"
	"github.com/tupyy/device-worker-ng/internal/scheduler/common"
	"github.com/tupyy/device-worker-ng/internal/scheduler/reconcile"
	"go.uber.org/zap"
)

const (
	defaultHeartbeatPeriod = 1 * time.Second
	gracefullShutdown      = 5 * time.Second
)

type evaluationResult struct {
	Active   bool
	Resource entity.Option[entity.CpuResource]
}

type Scheduler struct {
	// jobs holds all the current jobs
	jobs *Store
	// executor
	executor common.Executor
	// resource manager
	resourceManager common.ResourceManager
	// runCancel is the cancel function of the run goroutine
	runCancel context.CancelFunc
	// reconciler
	reconciler common.Reconciler
	// resource reconciler
	resourceReconciler common.ResourceReconciler
	// profileEvaluationResults holds the latest profile evaluation results received from profile manager
	profileEvaluationResults []profile.ProfileEvaluationResult
	// futures holds the future for each reconciliation function in progress
	futures map[string]*entity.Future[entity.Result[entity.JobState]]
	// resourceFutures holds the futures from resource reconciler
	resourceFutures map[string]*entity.Future[error]
	// runOnce prevents starting main goroute multiple times is _Start_ is called
	runOnce sync.Once
}

// New creates a new scheduler with the default heartbeat period of 2 seconds.
func New(executor common.Executor, resourcesEx common.ResourceManager) *Scheduler {
	return newScheduler(executor, resourcesEx, defaultHeartbeatPeriod)
}

// New creates a new scheduler with the hearbeat period provided by the user.
func NewWitHeartbeatPeriod(executor common.Executor, resourceManager common.ResourceManager, heartbeatPeriod time.Duration) *Scheduler {
	return newScheduler(executor, resourceManager, heartbeatPeriod)
}

func newScheduler(executor common.Executor, rm common.ResourceManager, heartbeatPeriod time.Duration) *Scheduler {
	return &Scheduler{
		jobs:                     NewStore(),
		executor:                 executor,
		resourceManager:          rm,
		reconciler:               reconcile.New(),
		resourceReconciler:       reconcile.NewResourceReconciler(),
		futures:                  make(map[string]*entity.Future[entity.Result[entity.JobState]]),
		resourceFutures:          make(map[string]*entity.Future[error]),
		profileEvaluationResults: make([]profile.ProfileEvaluationResult, 0),
	}
}

func (s *Scheduler) Start(ctx context.Context, input chan entity.Message, profileUpdateCh chan []profile.ProfileEvaluationResult) {
	// assure we run only once this function.
	s.runOnce.Do(func() {
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
	})
}

func (s *Scheduler) Stop(ctx context.Context) {
	zap.S().Info("shutting down scheduler")

	// shutdown goroutines
	s.runCancel()

	zap.S().Info("scheduler shutdown")
}

func (s *Scheduler) run(ctx context.Context, input chan entity.Option[[]entity.Workload], profileCh chan []profile.ProfileEvaluationResult) {
	executionSync := make(chan struct{}, 1)
	resourceSync := make(chan struct{}, 1)
	evaluationSync := make(chan struct{}, 1)

	// advanceTo does not block
	advanceTo := func(ch chan struct{}) {
		select {
		case ch <- struct{}{}:
		default:
		}
	}

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
			}
			// evaluate the new jobs with the latest profile evaluation results
			select {
			case evaluationSync <- struct{}{}:
			default:
			}
			// remove job which are not found in the EdgeWorkload manifest
			for _, j := range jobsToRemove {
				j.MarkForDeletion()
				j.SetTargetState(entity.ExitedState)
			}
		case results := <-profileCh:
			if reflect.DeepEqual(s.profileEvaluationResults, results) {
				break
			}
			s.profileEvaluationResults = results
			advanceTo(evaluationSync)
		case <-executionSync:
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
						future.CancelFunc()
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

				// remove the job only if it is markedForDeletion and it is stopped otherwise continue. The target state should be ExitedState.
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
						continue
					}
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
				reconcileCtx, cancel := context.WithCancel(ctx)
				future := s.reconciler.Reconcile(reconcileCtx, j, s.executor)
				future.CancelFunc = cancel
				s.futures[j.ID()] = future
			}
			// sync the resources
			advanceTo(resourceSync)
		case <-resourceSync:
			for _, j := range s.jobs.ToList() {
				// skip jobs that don't have target resources set
				if j.TargetResources().Equal(entity.CpuResource{}) {
					continue
				}

				if future, found := s.resourceFutures[j.ID()]; found {
					if err, isResolved := future.Poll(); isResolved {
						if err != nil {
							zap.S().Errorw("failed to reconcile resources", "job_id", j.ID(), "error", err)
						} else {
							zap.S().Infow("resources reconciled", "job_id", j.ID(), "resources", j.TargetResources())
							j.SetTargetResources(j.TargetResources())
						}
						future.CancelFunc()
						delete(s.resourceFutures, j.ID())
					}
				}

				// only jobs with running, not marked for deletion or k8s jobs are synced
				if j.CurrentState() != entity.RunningState || j.IsMarkedForDeletion() || j.Workload().Kind() == entity.K8SKind {
					continue
				}

				// get resources from system
				resources, err := s.getJobResources(context.TODO(), j)
				if err != nil {
					zap.S().Errorw("failed to read resources", "job_id", j.ID(), "error", err)
					continue
				}
				j.SetCurrentResources(resources)

				// reconcile resources only if current <> target
				if j.CurrentResources().Equal(j.TargetResources()) {
					continue
				}

				// reconcile resources
				reconcileCtx, cancel := context.WithCancel(ctx)
				future := s.resourceReconciler.Reconcile(reconcileCtx, j, s.resourceManager)
				future.CancelFunc = cancel
				s.resourceFutures[j.ID()] = future
			}
		case <-evaluationSync:
			zap.S().Infow("start evaluating job", "profile evaluation result", s.profileEvaluationResults)
			for _, j := range s.jobs.ToList() {
				// don't evaluate job marked for deletion
				if j.IsMarkedForDeletion() {
					continue
				}
				result := s.evaluate(j, s.profileEvaluationResults)
				switch result.Active {
				case true:
					zap.S().Infow("job's profiles evaluated to true", "job_id", j.ID(), "job_profiles", j.Workload().Profiles(), "profile_evaluation_result", s.profileEvaluationResults)
					j.SetTargetState(entity.RunningState)
					if !result.Resource.None {
						j.SetTargetResources(result.Resource.Value)
					}
				case false:
					zap.S().Infow("job's profiles evaluated to false", "job_id", j.ID(), "job_profiles", j.Workload().Profiles(), "profile_evaluation_result", s.profileEvaluationResults)
					j.SetTargetState(entity.InactiveState)
				}
			}
			// sync resources
			advanceTo(resourceSync)
		case <-heartbeat.C:
			executionSync <- struct{}{}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) GetWorkloadsStatus() map[string]entity.JobState {
	status := make(map[string]entity.JobState)
	for _, j := range s.jobs.ToList() {
		var name string
		switch j.Workload().Kind() {
		case entity.PodKind:
			pod, _ := j.Workload().(entity.PodWorkload)
			name = pod.Name
		default:
			continue
		}
		status[name] = j.CurrentState()
	}
	return status
}

func (s *Scheduler) createJob(w entity.Workload) (*entity.Job, error) {
	builder := entity.NewBuilder(w)
	if w.Cron() != "" {
		builder.WithCron(w.Cron())
	} else {
		builder.WithConstantRetry(10 * time.Second)
	}

	// add pre-set current state hook to compute next retry
	// if the job needs to be restarted
	builder.AddHook(entity.PostSetCurrentState, func(j *entity.Job, s entity.JobState) {
		if j.ShouldRestart() && j.Retry() != nil {
			if !j.Retry().MarkedForRestart {
				j.Retry().ComputeNext()
				j.Retry().MarkedForRestart = true
				zap.S().Debugw("marked job for restart", "job_id", j.ID(), "next_restart_after", j.Retry().Next())
			}
		}
		if s == entity.RunningState && j.Retry() != nil {
			j.Retry().MarkedForRestart = false
		}
	}).AddHook(entity.PostSetTargetState, func(j *entity.Job, s entity.JobState) {
		zap.S().Infow("target state set", "job_id", j.ID(), "target_state", s)
	})

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

func (s *Scheduler) evaluate(j *entity.Job, results []profile.ProfileEvaluationResult) evaluationResult {
	if len(j.Workload().Profiles()) == 0 || len(results) == 0 {
		return evaluationResult{
			Active:   true,
			Resource: entity.Option[entity.CpuResource]{None: true},
		}
	}

	/* map to hold evaluation results
	the key is the profile name and the value is a list of evaluation results for each profile's condition
	*/
	resultsMap := make(map[string][]*entity.Tuple[string, evaluationResult])

	// we populate the map with not active evaluation results.
	for _, p := range j.Workload().Profiles() {
		resultsMap[p.Name] = make([]*entity.Tuple[string, evaluationResult], 0, len(p.Conditions))
		for _, condition := range p.Conditions {
			conditionEv := entity.Tuple[string, evaluationResult]{
				Value1: condition.Name,
				Value2: evaluationResult{
					Active: false,
					Resource: entity.Option[entity.CpuResource]{
						None:  true,
						Value: entity.CpuResource{},
					},
				},
			}
			if condition.CPU != nil {
				conditionEv.Value2.Resource.None = false
				conditionEv.Value2.Resource.Value = entity.CpuResource{
					Value1: uint64(*condition.CPU),
					Value2: 100000,
				}
			}
			resultsMap[p.Name] = append(resultsMap[p.Name], &conditionEv)
		}
	}

	findFn := func(name string, list []*entity.Tuple[string, evaluationResult]) *entity.Tuple[string, evaluationResult] {
		for i := 0; i < len(list); i++ {
			elem := list[i]
			if elem.Value1 == name {
				return elem
			}
		}
		return &entity.Tuple[string, evaluationResult]{}
	}

	// // evaluate each condition of each profile
	for i := 0; i < len(results); i++ {
		result := results[i]
		profileResults := resultsMap[result.Name]
		for j := 0; j < len(result.ConditionsResults); j++ {
			condition := result.ConditionsResults[j]
			er := findFn(condition.Name, profileResults) // should be ok
			// do not evaluate the profile if there is an error.
			if condition.Error != nil {
				continue
			}
			if condition.Value {
				er.Value2.Active = true
			}
		}
	}

	// for job to be evaluated to true it needs that a least one condition per profile to be true
	// the resources with min CPU will be returned
	minCpu := int64(100000)
	resource := entity.Option[entity.CpuResource]{
		None:  true,
		Value: entity.CpuResource{},
	}
	// sum holds the number of active profiles. if sum >= len(profiles) than job is evaluated to true
	sum := 0
	for _, evaluationResults := range resultsMap {
		foundActive := false
		for _, evaluationResult := range evaluationResults {
			if evaluationResult.Value2.Active {
				foundActive = true
				if evaluationResult.Value2.Resource.None {
					continue
				}
				if evaluationResult.Value2.Resource.Value.Value1 <= uint64(minCpu) {
					resource = evaluationResult.Value2.Resource
					minCpu = int64(resource.Value.Value1)
				}
			}
		}
		if foundActive {
			sum++
		}
	}

	// for now, we consider that every profile has a resource defined
	e := evaluationResult{
		Active:   sum >= len(results),
		Resource: resource,
	}
	zap.S().Debugw("job evaluation result", "job_id", j.ID(), "result", e)
	return e
}

func (s *Scheduler) getJobResources(ctx context.Context, job *entity.Job) (entity.CpuResource, error) {
	pattern := fmt.Sprintf("%s", strings.ReplaceAll(job.ID(), "-", "_"))
	cgroup, err := s.resourceManager.GetCGroup(ctx, regexp.MustCompile(pattern), true)
	if err != nil {
		return entity.CpuResource{}, err
	}

	// strip the mountpoint /sys/fs/cgroup from path
	parts := strings.Split(cgroup, "/")
	cg := fmt.Sprintf("/%s", strings.Join(parts[4:], "/"))
	if cg == "" {
		return entity.CpuResource{}, fmt.Errorf("failed to find cgroup for job '%s'", job.ID())
	}

	cpu, err := s.resourceManager.GetResources(ctx, cg)
	if err != nil {
		if errors.Is(err, resources.ErrCPUMaxFileNotFound) {
			return cpu, nil
		}
	}
	return cpu, err
}
