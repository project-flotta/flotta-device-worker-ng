package reconcile

// type retryHandler struct {
// 	// nextSyncFunc holds the next sync function to be called
// 	nextSyncFunc syncFunc
// }

// func (r *retryHandler) wrap(s syncFunc) syncFunc {
// 	r.nextSyncFunc = s
// 	return r.sync
// }

// func (r *retryHandler) sync(ctx context.Context, j *job.DefaultJob, executor common.Executor) error {
// 	status, err := executor.GetState(context.TODO(), j.Workload())
// 	if err != nil {
// 		return err
// 	}

// 	state := job.NewState(status)
// 	j.SetCurrentState(state)

// 	if j.Retry() == nil {
// 		return r.nextSyncFunc(ctx, j, executor)
// 	}

// 	if j.CurrentState() != j.TargetState() && j.Retry().NextRetry.After(time.Now()) {
// 		zap.S().Debugf("cannot reconcile the job yet. Wait until '%s'", j.Retry().NextRetry)
// 		return nil
// 	}

// 	if j.CurrentState().OneOf(job.ExitedState, job.UnknownState) && j.TargetState() == job.RunningState {
// 		j.Retry().ComputeNextRetryTime()
// 		zap.S().Infow("job restarted", "job_id", j.ID(), "next retry after", j.Retry().NextRetry)
// 	}

// 	err = r.nextSyncFunc(ctx, j, executor)
// 	return err
// }
