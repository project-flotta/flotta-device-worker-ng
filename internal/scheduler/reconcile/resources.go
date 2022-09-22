package reconcile

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/tupyy/device-worker-ng/internal/entity"
	"github.com/tupyy/device-worker-ng/internal/scheduler/common"
	"go.uber.org/zap"
)

type resourcesSyncFunc func(ctx context.Context, job *entity.Job, executor common.ResourceManager) error

type resourceReconciler struct {
	syncFuncs map[entity.WorkloadKind]resourcesSyncFunc
	rootSlice string
}

func NewResourceReconciler() *resourceReconciler {
	r := &resourceReconciler{
		syncFuncs: make(map[entity.WorkloadKind]resourcesSyncFunc),
	}
	r.syncFuncs[entity.PodKind] = createPodResourceReconciliationFunc()
	return r
}

func (r *resourceReconciler) Reconcile(ctx context.Context, job *entity.Job, ex common.ResourceManager) *entity.Future[error] {
	fn, ok := r.syncFuncs[job.Workload().Kind()]
	if !ok {
		zap.S().Error("job kind not supported")
	}

	ch := make(chan error)
	go func(ctx context.Context, ch chan error, job *entity.Job, ex common.ResourceManager, fn resourcesSyncFunc) {
		err := fn(ctx, job, ex)
		ch <- err
		close(ch)
	}(ctx, ch, job, ex, fn)
	return entity.NewFuture(ch)
}

func createPodResourceReconciliationFunc() resourcesSyncFunc {
	return func(ctx context.Context, job *entity.Job, executor common.ResourceManager) error {
		zap.S().Debugw("reconcile resources for job", "job_id", job.ID(), "current_resources", job.CurrentResources(), "target_resources", job.TargetResources())
		pattern := fmt.Sprintf("%s", strings.ReplaceAll(job.ID(), "-", "_"))
		cgroup, err := executor.GetCGroup(ctx, regexp.MustCompile(pattern), true)
		if err != nil {
			return err
		}

		if cgroup == "" {
			return fmt.Errorf("failed to find cgroup for job '%s'", job.ID())
		}
		// strip the mountpoint /sys/fs/cgroup from path
		parts := strings.Split(cgroup, "/")
		cg := strings.Join(parts[4:], "/")
		zap.S().Debugw("found cgroup", "cgroup", cg)

		// set resources
		if err := executor.Set(ctx, path.Join("/", cg), job.TargetResources()); err != nil {
			return err
		}
		return nil
	}
}
