package scheduler_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler"
)

var _ = Describe("task test", func() {
	var task *scheduler.Task
	BeforeEach(func() {
		task = scheduler.NewTask("dummy", dummyWorkload("test"))
	})

	It("task should mutate correctly", func() {
		Expect(task.CurrentState()).To(Equal(scheduler.TaskStateReady))

		task.MutateTo(scheduler.TaskStateDeploying)

		Expect(task.CurrentState()).To(Equal(scheduler.TaskStateDeploying))

		task.MutateTo(scheduler.TaskStateDeployed)
		Expect(task.CurrentState()).To(Equal(scheduler.TaskStateDeployed))

		task.MutateTo(scheduler.TaskStateRunning)
		Expect(task.CurrentState()).To(Equal(scheduler.TaskStateRunning))

		task.MutateTo(scheduler.TaskStateExited)
		Expect(task.CurrentState()).To(Equal(scheduler.TaskStateExited))
	})
})

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
