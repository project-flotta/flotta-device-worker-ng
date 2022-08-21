package scheduler_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler"
	"github.com/tupyy/device-worker-ng/internal/scheduler/task"
)

var _ = Describe("test scheduler", func() {
	var (
		mockExecutor *scheduler.MockExecutor
		message      entity.Message
		emptyMessage entity.Message
		s            *scheduler.Scheduler
		input        chan entity.Message
		profileCh    chan entity.Message
	)

	BeforeEach(func() {
		mockExecutor = scheduler.NewMockExecutor()
		s = scheduler.New(mockExecutor)
		input = make(chan entity.Message, 1)
		profileCh = make(chan entity.Message)
		message = entity.Message{
			Kind: entity.WorkloadConfigurationMessage,
			Payload: entity.Option[[]entity.Workload]{
				Value: []entity.Workload{
					entity.PodWorkload{
						Name:          "workload1",
						Specification: "test",
					},
				},
			},
		}

		emptyMessage = entity.Message{
			Kind: entity.WorkloadConfigurationMessage,
			Payload: entity.Option[[]entity.Workload]{
				None: true,
			},
		}
	})

	AfterEach(func() {
		s.Stop(context.Background())
	})

	It("test workload from ready to stopped", func() {
		s.Start(context.Background(), input, profileCh)

		input <- message
		<-time.After(10 * time.Second)
		Expect(mockExecutor.RunCount).To(Equal(1))
		Expect(mockExecutor.StopCount).To(Equal(0))
		mockExecutor.SendStateToTask("workload1", task.RunningState, false)

		// remove workloads
		<-time.After(2 * time.Second)
		input <- emptyMessage
		<-time.After(10 * time.Second)
		Expect(mockExecutor.StopCount).To(Equal(1))
	})

	It("test workloads restart from exited", func() {
		s.Start(context.Background(), input, profileCh)

		input <- message
		<-time.After(10 * time.Second)
		Expect(mockExecutor.RunCount).To(Equal(1))
		Expect(mockExecutor.StopCount).To(Equal(0))
		mockExecutor.SendStateToTask("workload1", task.RunningState, false)

		// transition to exited
		<-time.After(2 * time.Second)
		mockExecutor.SendStateToTask("workload1", task.ExitedState, false)
		<-time.After(5 * time.Second)
		Expect(mockExecutor.RunCount).To(Equal(2))
	})

	It("test workloads which exit right after deployed", func() {
		s.Start(context.Background(), input, profileCh)

		input <- message
		<-time.After(5 * time.Second)
		Expect(mockExecutor.RunCount).To(Equal(1))
		Expect(mockExecutor.StopCount).To(Equal(0))
		mockExecutor.SendStateToTask("workload1", task.ExitedState, false)
		<-time.After(5 * time.Second)
		// expect to be restarted
		Expect(mockExecutor.RunCount).To(Equal(2))
	})

	It("test workloads which fail more than 3times and remains in exit", func() {
		s.Start(context.Background(), input, profileCh)

		input <- message
		<-time.After(5 * time.Second)
		Expect(mockExecutor.RunCount).To(Equal(1))
		Expect(mockExecutor.StopCount).To(Equal(0))

		// first failure
		mockExecutor.SendStateToTask("workload1", task.ExitedState, false)
		<-time.After(5 * time.Second)
		Expect(mockExecutor.RunCount).To(Equal(2))

		// 2nd failure
		mockExecutor.SendStateToTask("workload1", task.ExitedState, false)
		<-time.After(5 * time.Second)
		Expect(mockExecutor.RunCount).To(Equal(3))

		// 3rd failure
		mockExecutor.SendStateToTask("workload1", task.ExitedState, false)
		<-time.After(5 * time.Second)
		Expect(mockExecutor.RunCount).To(Equal(4))

		// 4rd failure
		mockExecutor.SendStateToTask("workload1", task.ExitedState, false)
		<-time.After(5 * time.Second)

		Expect(mockExecutor.RunCount).To(Equal(4))
	})

	It("test workloads which are modified by the operator", func() {
		s.Start(context.Background(), input, profileCh)

		message = entity.Message{
			Kind: entity.WorkloadConfigurationMessage,
			Payload: entity.Option[[]entity.Workload]{
				Value: []entity.Workload{
					entity.PodWorkload{
						Name:          "workload1",
						Specification: "test",
					},
				},
			},
		}

		input <- message
		<-time.After(10 * time.Second)
		Expect(mockExecutor.RunCount).To(Equal(1))
		Expect(mockExecutor.StopCount).To(Equal(0))

		message = entity.Message{
			Kind: entity.WorkloadConfigurationMessage,
			Payload: entity.Option[[]entity.Workload]{
				Value: []entity.Workload{
					entity.PodWorkload{
						Name:          "workload1",
						Specification: "other spec",
					},
				},
			},
		}
		input <- message
		<-time.After(10 * time.Second)
		Expect(mockExecutor.RunCount).To(Equal(2))
		Expect(mockExecutor.StopCount).To(Equal(1))
	})
})
