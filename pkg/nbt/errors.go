package nbt

import "fmt"

type errUnexpectedStatus struct {
	task *taskEntry
}

func (err *errUnexpectedStatus) Error() string {
	return fmt.Sprintf(`unexpected state %q for task`, err.task.status.String())
}

/* Error used when a task panics during execution. */
type errPanicked struct {
	panicErr interface{}
	task     *taskEntry
}

func (err *errPanicked) Error() string {
	return fmt.Sprintf("task %#v panicked: %v", err.task, err.panicErr)
}
