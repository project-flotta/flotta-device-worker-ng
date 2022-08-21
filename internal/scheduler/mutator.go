package scheduler

/*
	Mutate tries to find what is the next state based on marks and/or current state.
*/
// func (m *mutator) Mutate(t *Task) bool {
// 	// process task with marks
// 	for _, mark := range t.GetMarks() {
// 		switch mark {
// 		case mutateMark:
// 			val, ok := t.GetMark(mark)
// 			if !ok {
// 				zap.S().Warnw("mutation value not found", "task_id", t.ID(), "mark", mark)
// 				return false
// 			}
// 			// cannot deploy from inactive state
// 			if t.CurrentState() == TaskStateInactive {
// 				t.RemoveMark(mutateMark)
// 				zap.S().Debugw("task cannot be restarted from inactive state", "task_id", t.ID())
// 				return false
// 			}
// 			// if the task is exited or in unknown state and cannot be restarted.
// 			if t.CurrentState().OneOf(TaskStateExited, TaskStateUnknown) && !m.canRestart(t) {
// 				t.RemoveMark(mutateMark)
// 				zap.S().Warnw("task cannot be restarted because of too many failures. It will be transitioned to inactive", "task_id", t.ID())
// 				t.SetNextState(TaskStateInactive)
// 				return true
// 			}
// 			t.SetNextState(val.(TaskState))
// 			t.RemoveMark(mutateMark)
// 			return true
// 		case stopMark:
// 			if !t.CurrentState().OneOf(TaskStateDeploying, TaskStateDeployed, TaskStateRunning) {
// 				zap.S().Errorw("transition to exit is allowed only from deploying, deployed or running state", "task_id", t.ID())
// 				return false
// 			}
// 			t.SetNextState(TaskStateStopping)
// 			t.RemoveMark(stopMark)
// 			return true
// 		case inactiveMark:
// 			if !t.CurrentState().OneOf(TaskStateReady, TaskStateExited, TaskStateUnknown) {
// 				zap.S().Errorw("transition to inactive is allowed only from ready, exited or unknown state", "task_id", t.ID())
// 				return false
// 			}
// 			// transition to inactive is permitted only from ready, exit or unknown state.
// 			// a running job must be stopped before make it transition to inactive
// 			t.SetNextState(TaskStateInactive)
// 			return true
// 		}
// 	}
// 	return false
// }
