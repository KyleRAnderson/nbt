package nbt

import (
	"fmt"

	"gitlab.com/kyle_anderson/go-utils/pkg/queue"
)

func newTaskManager() *taskManager {
	return &taskManager{make(map[uintptr][]*taskEntry), 0, 0, queue.NewLinkedListQueue[*taskEntry]()}
}

type taskManager struct {
	registry                 map[uintptr][]*taskEntry
	numWaiting, numExecuting uint
	taskQueue                queue.Queue[*taskEntry]
}

func (tm *taskManager) processCompleteTask(task *taskEntry) {
	task.status = statusComplete
	for _, dependent := range task.dependents {
		dependent.dependencies.Remove(task)
		tm.processWaitingTask(dependent)
	}
}

func (tm *taskManager) processWaitingTask(task *taskEntry) {
	task.status = statusWaiting
	if task.IsReady() {
		tm.enqueue(task)
	}
}

/* Enqueues a task for execution. */
func (tm *taskManager) enqueue(task *taskEntry) {
	tm.taskQueue.Enqueue(task)
}

func (tm *taskManager) processRequirement(dependent *taskEntry, dependencies []Task) {
	for _, dependency := range dependencies {
		resolvedDependency := tm.resolve(dependency)
		if resolvedDependency.status != statusComplete {
			dependent.dependencies.Add(resolvedDependency)
		}
		resolvedDependency.dependents = append(resolvedDependency.dependents, dependent)
		if resolvedDependency.IsReady() && resolvedDependency.status == statusNew {
			/* Need to check that the task status is new to prevent adding multiple queue entries for the same task. */
			tm.enqueue(resolvedDependency)
		}
	}
}

func (tm *taskManager) resolve(task Task) (currentInstance *taskEntry) {
	key := task.Hash()
	var taskChain []*taskEntry
	if existingTasks, ok := tm.registry[key]; ok {
		taskChain = existingTasks
	} else {
		taskChain = make([]*taskEntry, 0, 1)
	}
	/* We may append to the taskChain, so in either case of the if statement, we should
	update the registry entry at the end. */
	defer func() { tm.registry[key] = taskChain }()
	found := false
	for _, entry := range taskChain {
		if task.Matches(entry.Task) {
			found = true
			currentInstance = entry
			break
		}
	}
	if !found {
		currentInstance = newTaskEntry(task)
		taskChain = append(taskChain, currentInstance)
	}
	return
}

type errUnexpectedStatus struct {
	task *taskEntry
}

func (err *errUnexpectedStatus) Error() string {
	return fmt.Sprintf(`unexpected state %q for task`, err.task.status.String())
}

/* Runs the given task. */
func (tm *taskManager) run(task *taskEntry, comms *supervisorComms) {
	switch task.status {
	case statusNew:
		task.handler = newChanHandler[*taskEntry]()
		go func() {
			defer close(task.handler.messages)
			task.Perform(task.handler)
		}()
	case statusWaiting:
		tm.numWaiting--
	default:
		panic(&errUnexpectedStatus{task})
	}
	task.status = statusRunning
	go superviseTask[*taskEntry](task, task.handler, comms)
	tm.numExecuting++
}

func dependencyQueueSize(maxParallelTasks uint) uint {
	return 4 * maxParallelTasks
}

func (manager *taskManager) execute(mainTask Task, maxParallelTasks uint) {
	if maxParallelTasks <= 0 {
		panic("numJobs must be positive!")
	}
	/* No need to close these channels since it wouldn't signal anything anyway. */
	comms := supervisorComms{
		messages:        make(chan messenger[*taskEntry], dependencyQueueSize(maxParallelTasks)),
		resolutionQueue: make(chan resolveRequester, maxParallelTasks),
	}

	manager.run(manager.resolve(mainTask), &comms)

	for manager.numExecuting > 0 {
		select {
		case message := <-comms.messages:
			if status := message.RequestedStatus(); status != nil {
				switch *status {
				case statusComplete:
					manager.numExecuting--
					manager.processCompleteTask(message.Subject())
				case statusWaiting:
					manager.numWaiting++
					manager.numExecuting--
					manager.processWaitingTask(message.Subject())
				default:
					// Do nothing
				}
			}
			if dependencies := message.Dependencies(); dependencies != nil {
				manager.processRequirement(message.Subject(), dependencies)
			}
		case request := <-comms.resolutionQueue:
			/* This will not block with the implementation of chanMessageCallbacks that we have, since
			only one item will ever get placed on the callback channel, and it is a buffered channel. */
			request.Callback() <- manager.resolve(request.ToResolve())
		}
	}

	// TODO handle deadlock, which should be indicated if manager.numExecuting <= 0 && manager.numWaiting > 0
}
