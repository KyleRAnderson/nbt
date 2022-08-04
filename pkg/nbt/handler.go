package nbt

/* Handlers make requests on behalf of workers, and relay received information. */

/* A handler that uses channels for all of its operations. */
type chanHandler[T Task] struct {
	messages     chan handlerMessenger
	resolveQueue chan resolveRequester
	/* A channel that is waited on when the task requests to wait.
	This channel should be written to to signal that this task should resume execution. */
	waiter chan struct{}
}

func newChanHandler[T Task]() *chanHandler[T] {
	messages := make(chan handlerMessenger, 4)
	resolveQueue := make(chan resolveRequester)
	waiter := make(chan struct{})
	return &chanHandler[T]{messages, resolveQueue, waiter}
}

func (h *chanHandler[T]) Require(task Task) {
	h.messages <- &dependencyDeclaration{dependencies: []Task{task}}
}

func (h *chanHandler[T]) Wait() {
	h.messages <- statusUpdate{newStatus: statusWaiting}
	<-h.waiter
}

func (h *chanHandler[T]) Resolve(task Task) Task {
	request, resolution := newResolveRequest(task)
	h.resolveQueue <- request
	return <-resolution
}
