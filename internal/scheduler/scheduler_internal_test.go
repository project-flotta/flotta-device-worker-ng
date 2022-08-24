package scheduler

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/task"
)

func TestAdvanceToState(t *testing.T) {
	g := NewWithT(t)
	myTask := task.NewDefaultTask("1", entity.PodWorkload{})
	ex := NewMockExecutor()
	s := New(ex)

	// s.advanceToState(myTask, task.RunningState, task.MarkBasedEdgeType)
	// g.Expect(myTask.CurrentState()).To(Equal(task.DeployingState))

	myTask.SetCurrentState(task.DeployedState)
	s.advanceToState(myTask, task.ExitedState, task.EventBasedEdgeType)
	g.Expect(myTask.CurrentState()).To(Equal(task.RunningState))
}
