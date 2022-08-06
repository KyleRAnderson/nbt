package nbt

import (
	"gitlab.com/kyle_anderson/go-utils/pkg/set"
)

/* TODO would like it if the taskEntry struct could have some real private members.
Status should only be set through the setStatus method. */
type taskEntry struct {
	Task
	/* Slice of tasks that are dependent and still waiting on this task. */
	dependents []*taskEntry
	/* Tasks upon which this task depends. */
	dependencies set.Set[*taskEntry]
	status       taskStatus
	handler      *chanHandler[*taskEntry]

	onWaitingHooks []func(*taskEntry)
}

func newTaskEntry(t Task) *taskEntry {
	return &taskEntry{
		Task:         t,
		dependents:   make([]*taskEntry, 0),
		dependencies: set.NewComparable[*taskEntry](),
		status:       statusNew,
	}
}

/* Returns true if this task is ready to execute, when all of its dependencies have been met, false otherwise. */
func (te *taskEntry) IsReady() bool {
	return te.dependencies.Size() <= 0
}

/* Adds a callback to be executed when this task next goes into the waiting state.
If the task is currently in the waiting state, then the callback won't be executed until the task
exits this state and re-enters it.
The callback is only called once after a state update. Consumers wishing to have the callback
fired for subsequent updates should re-subscribe during the callback itself. */
func (te *taskEntry) onWaiting(callback func(*taskEntry)) {
	te.onWaitingHooks = append(te.onWaitingHooks, callback)
}

func (te *taskEntry) fireCallbacks(callbacks *[]func(*taskEntry)) {
	originalCallbacks := *callbacks
	/* Important to set the callbacks slice to nil to avoid iterating
	over a slice that is being mutated, which could happen when a callback
	is added during a callback. */
	*callbacks = nil
	for _, callback := range originalCallbacks {
		callback(te)
	}
}

type taskStatus uint

const (
	statusNew taskStatus = iota
	statusRunning
	statusWaiting
	statusComplete
	/* The task encountered an error and stopped executing. */
	statusErrored
)

func (ts taskStatus) String() (statusName string) {
	switch ts {
	case statusNew:
		statusName = "New"
	case statusRunning:
		statusName = "Running"
	case statusWaiting:
		statusName = "Waiting"
	case statusComplete:
		statusName = "Done"
	case statusErrored:
		statusName = "Errored"
	default:
		statusName = "ERROR - UNKNOWN STATUS"
	}
	return
}
