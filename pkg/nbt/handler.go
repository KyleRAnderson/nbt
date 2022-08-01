package nbt

/* A handler that uses channels for all of its operations. */
type chanHandler[T Task] struct {
	requireQueue chan Task
	resolveQueue chan resolveRequest
	waitRequests chan struct{}
	/* A channel that is waited on when the task requests to wait.
	This channel should be written to to signal that this task should resume execution. */
	waiter chan struct{}
}

func newChanHandler[T Task]() *chanHandler[T] {
	waitRequests := make(chan struct{})
	requireQueue := make(chan Task, 4)
	resolveQueue := make(chan resolveRequest)
	waiter := make(chan struct{})
	return &chanHandler[T]{requireQueue, resolveQueue, waitRequests, waiter}
}

func (h *chanHandler[T]) Require(task Task) {
	h.requireQueue <- task
}

func (h *chanHandler[T]) Wait() {
	h.waitRequests <- struct{}{}
	<-h.waiter
}

func (h *chanHandler[T]) Resolve(task Task) Task {
	resolution := make(chan Task, 1)
	h.resolveQueue <- resolveRequest{resolution, task}
	return <-resolution
}
