package nbt

/* Interface for message types sent to a task handler.
These message types do not have the task making the request included,
as the handler holds that data. */
type handlerMessenger interface {
	/* New dependency declarations. Can be nil or empty. */
	Dependencies() []Task
	/* Status being requested by the task. Nil to indicate that no status update is being requested.
	Request may be denied. */
	RequestedStatus() *taskStatus
	/* Any sort of error that has occurred. Having an error alone does not mark the task as
	having failed, for that, the status must also be updated to statusErrored. */
	Error() error
}

/* Interface for message types sent to the manager, which need the task which made the
request present to be actionable. This is what the subject is for. */
type messenger[T Task] interface {
	handlerMessenger
	subjectHolder[T]
}

type subjectHolder[T Task] interface {
	Subject() T
}

type messageWithSubject[T Task] struct {
	task T
	handlerMessenger
}

func (mws *messageWithSubject[T]) Subject() T { return mws.task }

func addSubject[T Task](subject T, message handlerMessenger) messenger[T] {
	return &messageWithSubject[T]{subject, message}
}

type blankMessage struct{}

func (blankMessage) Dependencies() []Task         { return nil }
func (blankMessage) RequestedStatus() *taskStatus { return nil }
func (blankMessage) Error() error                 { return nil }

type statusUpdate struct {
	newStatus taskStatus
	blankMessage
}

func (su statusUpdate) RequestedStatus() *taskStatus {
	/* Notice that a copy of the statusUpdate message is made, so we are not returning
	a pointer that can mutate the original object. */
	return &su.newStatus
}

type dependencyDeclaration struct {
	dependencies []Task
	blankMessage
}

func (r *dependencyDeclaration) Dependencies() []Task { return r.dependencies }

type errorMessage struct {
	err error
	blankMessage
}

func (em *errorMessage) RequestedStatus() *taskStatus {
	status := statusErrored
	return &status
}

func (em *errorMessage) Error() error { return em.err }

/* Message type used when resolution of a task is requested. */
type resolveRequester interface {
	ToResolve() Task
	Callback() chan<- Task
}

// Make private
type resolveRequest struct {
	toResolve Task
	callback  chan<- Task
}

func (rr *resolveRequest) ToResolve() Task       { return rr.toResolve }
func (rr *resolveRequest) Callback() chan<- Task { return rr.callback }

func newResolveRequest(toResolve Task) (request resolveRequester, callback <-chan Task) {
	rawCallback := make(chan Task, 1)
	return &resolveRequest{toResolve, rawCallback}, rawCallback
}
