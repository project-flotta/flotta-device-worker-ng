package scheduler

import (
	"context"
	"errors"
	"time"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/containers"
	"github.com/tupyy/device-worker-ng/internal/scheduler/task"
	"go.uber.org/zap"
)

type actionType int

const (
	defaultHeartbeatPeriod = 1 * time.Second
	gracefullShutdown      = 5 * time.Second

	// action type
	runAction actionType = iota
	stopAction
	deleteAction

	// marks type
	deletionMark  string = "deletion"
	stopMark      string = "stop"
	deployMark    string = "deploy"
	inactiveMark  string = "inactive"
	nextStateMark string = "next_state"
)

//go:generate mockgen -package=scheduler -destination=mock_executor.go --build_flags=--mod=mod . Executor
type Executor interface {
	Run(ctx context.Context, w entity.Workload) *Future[task.State]
	Stop(ctx context.Context, w entity.Workload)
}

type Scheduler struct {
	// tasks holds all the current tasks
	tasks *containers.Store[task.Task]
	// futures holds the futures of executed tasks
	// the hash of the task is the key
	futures map[string]*Future[task.State]
	// executor
	executor Executor
	// runCancel is the cancel function of the run goroutine
	runCancel context.CancelFunc
	// executionQueue holds the tasks which must be executed by executor
	executionQueue *containers.ExecutionQueue[actionType, task.Task]
}

// New creates a new scheduler with the default heartbeat period of 2 seconds.
func New(executor Executor) *Scheduler {
	return newExecutor(executor, defaultHeartbeatPeriod)
}

// New creates a new scheduler with the hearbeat period provided by the user.
func NewWitHeartbeatPeriod(executor Executor, heartbeatPeriod time.Duration) *Scheduler {
	return newExecutor(executor, heartbeatPeriod)
}

func newExecutor(executor Executor, heartbeatPeriod time.Duration) *Scheduler {
	return &Scheduler{
		tasks:          containers.NewStore[task.Task](),
		futures:        make(map[string]*Future[task.State]),
		executionQueue: containers.NewExecutionQueue[actionType, task.Task](),
		executor:       executor,
	}
}

func (s *Scheduler) Start(ctx context.Context, input chan entity.Message, profileUpdateCh chan entity.Message) {
	runCtx, cancel := context.WithCancel(ctx)
	s.runCancel = cancel

	taskCh := make(chan entity.Option[[]entity.Workload])
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
					taskCh <- val
				}
			case <-ctx.Done():
				return
			}
		}
	}(runCtx)
	go s.run(runCtx, taskCh, profileUpdateCh)
}

func (s *Scheduler) Stop(ctx context.Context) {
	zap.S().Info("shutting down scheduler")

	// shutdown goroutines
	s.runCancel()

	zap.S().Info("scheduler shutdown")
}

func (s *Scheduler) run(ctx context.Context, input chan entity.Option[[]entity.Workload], profileCh chan entity.Message) {
	execution := make(chan struct{}, 1)
	doneExecutionCh := make(chan struct{})
	mutate := make(chan struct{}, 1)
	mark := make(chan struct{}, 1)

	heartbeat := time.NewTicker(defaultHeartbeatPeriod)

	for {
		select {
		case o := <-input:
			if o.None {
				s.removeTasks(s.tasks.ToList())
				break
			}
			// add tasks
			m := make(map[string]struct{}) // holds temporary the new tasks
			for _, w := range o.Value {
				m[w.Hash()] = struct{}{}
				task := task.NewTaskWithRetry(task.NewDefaultTask(w.ID(), w), 3)
				if oldTask, found := s.tasks.FindByName(task.Name()); found {
					if task.Equal(oldTask) {
						continue
					}
					// something changed in the workload. Stop the old one and start the new one
					zap.S().Infow("workload changed", "id", oldTask.Name)
					s.removeTask(oldTask)
				}
				s.tasks.Add(task)
			}
			// check if there are task removed
			it := s.tasks.Iterator()
			for it.HasNext() {
				task, _ := it.Next()
				if _, found := m[task.ID()]; !found {
					// task was removed from the manifest.
					zap.S().Infow("remove workload", "id", task.ID())
					s.removeTask(task)
				}
			}
		case <-mark:
			it := s.tasks.Iterator()
			for it.HasNext() {
				t, _ := it.Next()
				// poll his future if any
				if future, found := s.futures[t.ID()]; found {
					result, _ := future.Poll()
					if result.IsReady() {
						zap.S().Debugw("poll future", "id", t.ID(), "result", result)
						t.AddMark(nextStateMark, result.Value.String())
					}

					// future is resolved when task has either been stopped or exited.
					if future.Resolved() {
						zap.S().Debugw("future resolved", "id", t.ID())
						delete(s.futures, t.ID())
					}
					continue
				}
				// no future yet meaning the task has not been deployed yet or it exited.
				// first evaluate task. if true than deploy it.
				if s.evaluate(t) {
					t.AddMark(deployMark, deployMark)
				}
			}
			mutate <- struct{}{}
		case <-mutate:
			it := s.tasks.Iterator()
			for it.HasNext() {
				t, _ := it.Next()

				if mutated, err := s.mutate(t); err == nil && !mutated {
					continue
				} else if err != nil {
					zap.S().Errorw("failed to mutate task", "task_id", t.ID(), "error", err)
					continue
				}

				// resolve the mutations
				switch t.NextState() {
				case task.StoppingState:
					zap.S().Debugw("stop task", "id", t.Name())
					s.executionQueue.Push(stopAction, t)
				case task.DeployingState:
					zap.S().Debugw("deploy task", "id", t.Name())
					s.executionQueue.Push(runAction, t)
				case task.DeletionState:
					zap.S().Debugw("remove task", "id", t.Name())
					s.executionQueue.Push(deleteAction, t)
				default:
					t.SetCurrentState(t.NextState())
				}
			}
			if s.executionQueue.Size() > 0 {
				execution <- struct{}{}
			}
		case <-execution:
			// execute every task in the execution queue
			go s.execute(context.Background(), doneExecutionCh)
			// stop heartbeat while we are consuming the execution queue.
			// Once is done, reset the timer.
			heartbeat.Stop()
		case <-doneExecutionCh:
			heartbeat.Reset(defaultHeartbeatPeriod)
		case <-heartbeat.C:
			mark <- struct{}{}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) execute(ctx context.Context, doneCh chan struct{}) {
	for s.executionQueue.Size() > 0 {
		// stopping task has higher priority
		s.executionQueue.Sort(stopAction)
		action, t, err := s.executionQueue.Pop()
		if err != nil {
			zap.S().Errorw("failed to pop task from queue", "error", err)
			break
		}
		switch action {
		case stopAction:
			t.SetCurrentState(task.StoppingState)
			s.executor.Stop(context.Background(), t.Workload())
		case runAction:
			t.SetCurrentState(task.DeployingState)
			future := s.executor.Run(context.Background(), t.Workload())
			s.futures[t.ID()] = future
		case deleteAction:
			delete(s.futures, t.ID())
			s.tasks.Delete(t)
		}
		select {
		case <-ctx.Done():
			doneCh <- struct{}{}
			return
		default:
		}
	}
	doneCh <- struct{}{}
}

// evaluate evaluates task's profiles based on current device profile.
func (s *Scheduler) evaluate(t task.Task) bool {
	return true
}

func (s *Scheduler) removeTasks(tasks []task.Task) {
	for _, t := range tasks {
		s.removeTask(t)
	}
}

func (s *Scheduler) removeTask(task task.Task) {
	task.AddMark(stopMark, stopMark)
	task.AddMark(deletionMark, deletionMark)
}

// mutate process one mark at the time.
func (s *Scheduler) mutate(t task.Task) (bool, error) {
	if !t.HasMarks() {
		return false, nil
	}

	var (
		nextState task.State
		edgeType  task.EdgeType
	)

	mark := t.Peek()
	// task is marked. We try to mutate the task based on those marks.
	switch mark.Kind {
	case stopMark:
		nextState = task.StoppingState
		edgeType = task.MarkBasedEdgeType
	case deletionMark:
		nextState = task.DeletionState
		edgeType = task.MarkBasedEdgeType
	case inactiveMark:
		nextState = task.InactiveState
		edgeType = task.MarkBasedEdgeType
	case deployMark:
		nextState = task.DeployingState
		edgeType = task.MarkBasedEdgeType
	case nextStateMark:
		nextState = task.FromString(mark.Value)
		edgeType = task.EventBasedEdgeType
	}
	state, err := s.advanceToState(t, nextState, edgeType)
	if err != nil {
		return false, err
	}
	if state == nextState {
		t.PopMark()
	}
	if err := t.SetNextState(state); err != nil {
		return false, err
	}
	return true, nil
}

// advanceToState return the shortest path to 'state' walking only on edge whose type is 'edgeType'.
// each edge has a weight of 1.
func (s *Scheduler) advanceToState(t task.Task, state task.State, edgeType task.EdgeType) (task.State, error) {
	paths, err := t.FindState(state)
	if err != nil {
		return task.UnknownState, err
	}

	if len(paths) == 0 {
		return task.UnknownState, errors.New("no path found")
	}

	// loop though all the paths and see how far we can get walking *only* on the edge of type 'edgeType'
	// we stop at the last state which has edge of type 'edgeType'
	// the state with the lowest score is returned or error if we didn't find anything
	results := make(map[task.State]int) // we hold the task and the distance.
	for _, path := range paths {
		var (
			pathScore int
			state     task.State
		)
		// the first state is the starting point itself. We need to look at the next state and the current edge.
		for i := 0; i < len(path)-1; i++ {
			nextState := path[i+1].Node
			if path[i].Edge == edgeType {
				pathScore = i
				state = nextState
			}
		}
		if oldScore, ok := results[state]; ok {
			if oldScore > pathScore {
				results[state] = pathScore
			}
		} else {
			results[state] = pathScore
		}
	}
	var ss task.State
	min := 100
	for state, score := range results {
		if score < min {
			ss = state
			min = score
		}
	}
	return ss, nil
}
