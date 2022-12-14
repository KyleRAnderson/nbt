package nbt

/* Supervisors supervise workers, receiving information about them through their handlers.
Supervisors come and go as they only deal with running tasks. */

/* Interface to be implemented in order for the supervisor to communicate with the manager. */
type supervisorCommunicator[T Task] interface {
	SendMessage(T, handlerMessenger)
	RequestResolution(resolveRequester)
}

type supervisorComms struct {
	messages        chan messenger[*taskEntry]
	resolutionQueue chan resolveRequester
}

func (c *supervisorComms) SendMessage(task *taskEntry, message handlerMessenger) {
	c.messages <- addSubject(task, message)
}

func (c *supervisorComms) RequestResolution(r resolveRequester) {
	c.resolutionQueue <- r
}

/* Supervises the given task, interacting with `handler` to achieve this. */
func superviseTask[T Task](task T, handler *chanHandler[T], comms supervisorCommunicator[T]) {
	/* Use a type parameter to prevent this function from using members of *taskEntry. */
	for {
		select {
		case request := <-handler.resolveQueue:
			comms.RequestResolution(request)
		case message, isOpen := <-handler.messages:
			if !isOpen {
				comms.SendMessage(task, statusUpdate{newStatus: statusComplete})
				return
			} else {
				comms.SendMessage(task, message)
			}
			if newStatus := message.RequestedStatus(); newStatus != nil && shouldSupervisorExit(*newStatus) {
				/* A new supervisor is created whenever a task is scheduled, including when it is resumed, so we can
				safely exit here. */
				return
			}
		}
	}
}

func shouldSupervisorExit(status taskStatus) bool {
	switch status {
	case statusWaiting, statusComplete, statusErrored:
		return true
	default:
		return false
	}
}
