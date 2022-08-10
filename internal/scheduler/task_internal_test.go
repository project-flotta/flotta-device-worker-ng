package scheduler

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/tupyy/device-worker-ng/internal/entity"
)

func TestTask(t *testing.T) {
	g := NewWithT(t)
	task := NewTask("dummy", dummyWorkload("test"))

	g.Expect(task.CurrentState()).To(Equal(TaskStateReady))

	err := task.MarkForDeploy()
	g.Expect(err).To(BeNil())
	g.Expect(task.NextState()).To(Equal(TaskStateDeploying))

	task.TransitionTo(TaskStateDeploying)

	g.Expect(task.CurrentState()).To(Equal(TaskStateDeploying))

	task.TransitionTo(TaskStateDeployed)
	g.Expect(task.CurrentState()).To(Equal(TaskStateDeployed))

	task.TransitionTo(TaskStateRunning)
	g.Expect(task.CurrentState()).To(Equal(TaskStateRunning))

	task.TransitionTo(TaskStateExited)
	g.Expect(task.CurrentState()).To(Equal(TaskStateExited))
	g.Expect(task.failures).To(Equal(1))

	// expect error when set next state to TaskStateStopping
	err = task.MarkForStop()
	g.Expect(err).NotTo(BeNil())

	// expect nil when set next state to TaskStateDeploying
	err = task.MarkForDeploy()
	g.Expect(err).To(BeNil())
	g.Expect(task.CurrentState()).To(Equal(TaskStateExited))
	g.Expect(task.NextState()).To(Equal(TaskStateDeploying))
}

func TestTaskWhenSetEnableToFalse(t *testing.T) {
	g := NewWithT(t)
	task := NewTask("dummy", dummyWorkload("test"))

	g.Expect(task.CurrentState()).To(Equal(TaskStateReady))

	err := task.MarkForDeploy()
	g.Expect(err).To(BeNil())
	g.Expect(task.NextState()).To(Equal(TaskStateDeploying))

	task.TransitionTo(TaskStateDeploying)

	g.Expect(task.CurrentState()).To(Equal(TaskStateDeploying))

	task.TransitionTo(TaskStateDeployed)
	g.Expect(task.CurrentState()).To(Equal(TaskStateDeployed))

	task.TransitionTo(TaskStateRunning)
	g.Expect(task.CurrentState()).To(Equal(TaskStateRunning))
	task.Enable(false)

	g.Expect(task.NextState()).To(Equal(TaskStateStopping))
	g.Expect(task.IsEnabled()).To(BeFalse())
}

func TestTaskWhenSetEnableToTrue(t *testing.T) {
	g := NewWithT(t)
	task := NewTask("dummy", dummyWorkload("test"))
	task.Enable(false)
	g.Expect(task.IsEnabled()).To(BeFalse())

	g.Expect(task.CurrentState()).To(Equal(TaskStateReady))

	err := task.MarkForDeploy()
	g.Expect(err).To(BeNil())
	g.Expect(task.NextState()).To(Equal(TaskStateDeploying))

	task.TransitionTo(TaskStateDeploying)

	g.Expect(task.CurrentState()).To(Equal(TaskStateDeploying))

	task.TransitionTo(TaskStateDeployed)
	g.Expect(task.CurrentState()).To(Equal(TaskStateDeployed))

	task.TransitionTo(TaskStateRunning)
	task.TransitionTo(TaskStateStopped)

	g.Expect(task.CurrentState()).To(Equal(TaskStateStopped))

}

type dummyWorkload string

func (d dummyWorkload) ID() string {
	return string(d)
}

func (d dummyWorkload) Kind() entity.WorkloadKind {
	return entity.PodKind
}

func (d dummyWorkload) String() string {
	return string(d)
}

func (d dummyWorkload) Hash() string {
	return string(d)
}

func (d dummyWorkload) Profiles() []entity.WorkloadProfile {
	return []entity.WorkloadProfile{}
}
