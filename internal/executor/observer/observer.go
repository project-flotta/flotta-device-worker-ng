package observer

import (
	"context"
	"time"

	"github.com/tupyy/device-worker-ng/internal/executor/common"
	"github.com/tupyy/device-worker-ng/internal/scheduler/task"
	"go.uber.org/zap"
)

type Executor interface {
	List(ctx context.Context) ([]common.WorkloadInfo, error)
}

type workloadInfo struct {
	ID            string
	FutureChannel chan task.State
}

type Observer struct {
	workloads  map[Executor]map[string]chan task.State
	cancelFunc context.CancelFunc
}

func New() *Observer {
	ctx, cancel := context.WithCancel(context.Background())
	o := &Observer{
		workloads:  make(map[Executor]map[string]chan task.State),
		cancelFunc: cancel,
	}
	go o.run(ctx)
	return o
}

func (o *Observer) Shutdown() {
	o.cancelFunc()
}

func (o *Observer) RegisterWorkload(id string, ex Executor) chan task.State {
	ch := make(chan task.State)
	_, found := o.workloads[ex]
	if !found {
		o.workloads[ex] = make(map[string]chan task.State)
	}

	o.workloads[ex][id] = ch
	return ch
}

func (o *Observer) run(ctx context.Context) {
	timer := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-timer.C:
			for ex, w := range o.workloads {
				info, err := ex.List(context.TODO())
				zap.S().Debugw("list workloads", "info", info)
				if err != nil || info == nil {
					if len(w) > 0 {
						for _, ch := range w {
							ch <- task.ExitedState
							close(ch)
						}
						// cleanup everything
						delete(o.workloads, ex)
						break
					}
				}
				// range over all the workloads and try to find the info.
				// If not found, it means the task exited somehow than close the channel and remove the workload from map
				for id, ch := range w {
					found := false
					for _, i := range info {
						if i.Id == id {
							zap.S().Debugw("write to future", "state", i.Status)
							ch <- mapToState(i.Status)
							found = true
							break
						}
					}
					if !found {
						zap.S().Debugw("info not found", "task_id", id)
						ch <- task.ExitedState
						close(ch)
						delete(w, id)
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
