package nbt

import (
	"gitlab.com/kyle_anderson/go-utils/pkg/set"
)

type taskEntry struct {
	Task
	/* Slice of tasks that are dependent and still waiting on this task. */
	dependents []*taskEntry
	/* Tasks upon which this task depends. */
	dependencies set.Set[*taskEntry]
	status       taskStatus
	handler      *chanHandler[*taskEntry]
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

type taskStatus uint

const (
	statusNew taskStatus = iota
	statusRunning
	statusWaiting
	statusComplete
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
	default:
		statusName = "ERROR - UNKNOWN STATUS"
	}
	return
}
