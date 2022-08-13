package ntr

import "fmt"

/* Error type returned when a task is named in the arguments but cannot be found among the registered tasks. */
type ErrTaskNotFound struct {
	taskName string
}

func (tnf *ErrTaskNotFound) Error() string { return fmt.Sprintf("task %q not found", tnf.taskName) }

/* Error type returned if the task supplier for a registered task returns an error. */
type ErrTaskConstruction struct {
	taskName, arg string
	returnedErr   error
}

func (etc *ErrTaskConstruction) Error() string {
	return fmt.Sprintf("failed to construct task %q with arg %q: %v", etc.taskName, etc.arg, etc.returnedErr)
}
func (etc *ErrTaskConstruction) Unwrap() error { return etc.returnedErr }
