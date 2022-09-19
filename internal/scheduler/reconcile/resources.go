package reconcile

import (
	"context"
	"time"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/common"
	"go.uber.org/zap"
)

type resourcesSyncFunc func(ctx context.Context, job *entity.Job, executor common.ResourceManager)

type resourceReconciler struct {
	syncFuncs map[entity.WorkloadKind]resourcesSyncFunc
	rootSlice string
}

func NewResourceReconciler() *resourceReconciler {
	r := &resourceReconciler{
		syncFuncs: make(map[entity.WorkloadKind]resourcesSyncFunc),
	}
	return r
}

func (r *resourceReconciler) Reconcile(ctx context.Context, job *entity.Job, ex common.ResourceManager) *entity.Future[error] {
	zap.S().Debugw("resources reconciling started", "now", time.Now())
	// fn, ok := r.syncFuncs[job.Workload().Kind()]
	// if !ok {
	// 	zap.S().Error("job kind not supported")
	// }

	ch := make(chan error)
	go func(ctx context.Context, ch chan error, job *entity.Job, ex common.ResourceManager, fn resourcesSyncFunc) {
		switch job.TargetState() {
		case entity.RunningState:
			// check if the slice exists. if not create it
			if !ex.SliceExists(ctx, job.ID()) {
				if err := ex.CreateSlice(ctx, job.ID()); err != nil {
					ch <- err
					break
				}
			}
			if err := ex.Set(ctx, job.ID(), job.TargetResources()); err != nil {
				ch <- err
			}
		default: // for every other state, remove the slice
			if !ex.SliceExists(ctx, job.ID()) {
				break
			}
			if err := ex.RemoveSlice(ctx, job.ID()); err != nil {
				ch <- err
			}
		}
		close(ch)
	}(ctx, ch, job, ex, nil)
	return entity.NewFuture(ch)
}
