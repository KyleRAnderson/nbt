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
			} else {
				comms.SendMessage(task, message)
			}
			return
		}
	}
}
