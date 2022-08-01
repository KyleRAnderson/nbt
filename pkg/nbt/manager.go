package nbt

import (
	"fmt"

	"gitlab.com/kyle_anderson/go-utils/pkg/queue"
)

func newTaskManager() *taskManager {
	return &taskManager{make(map[int][]*taskEntry), 0, 0, queue.NewLinkedListQueue()}
}

type taskManager struct {
	registry                 map[int][]*taskEntry
	numWaiting, numExecuting uint
	taskQueue                queue.Queue[*taskEntry]
}

type managerComms struct {
	doneQueue chan *taskEntry
	/* Queue of requests for tasks to wait. */
	waitQueue chan *taskEntry
	/* Queue of dependency declarations. */
	dependencyQueue chan dependencyDeclaration[*taskEntry]
	resolutionQueue chan resolveRequest
}

func (c *managerComms) MarkDone(task *taskEntry) {
	c.doneQueue <- task
}
func (c *managerComms) MarkWaiting(task *taskEntry) {
	c.waitQueue <- task
}
func (c *managerComms) RequestResolution(r resolveRequest) {
	c.resolutionQueue <- r
}
func (c *managerComms) DeclareDependency(decl dependencyDeclaration[*taskEntry]) {
	c.dependencyQueue <- decl
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
		dependent.dependencies.Add(resolvedDependency)
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
func (tm *taskManager) run(task *taskEntry, comms *managerComms) {
	switch task.status {
	case statusNew:
		task.handler = newChanHandler[*taskEntry]()
		go func() {
			defer close(task.handler.waitRequests)
			task.Perform(task.handler)
		}()
	case statusWaiting:
		tm.numWaiting--
	default:
		panic(&errUnexpectedStatus{task})
	}
	task.status = statusRunning
	go monitorTask[*taskEntry](task, task.handler, comms)
	tm.numExecuting++
}

func monitorTask[T Task](task T, handler *chanHandler[T], comms interface {
	DeclareDependency(dependencyDeclaration[T])
	RequestResolution(resolveRequest)
	MarkDone(T)
	MarkWaiting(T)
}) {
	/* Use a type parameter to prevent this function from using members of *taskEntry. */
	for {
		select {
		case requirement := <-handler.requireQueue:
			comms.DeclareDependency(dependencyDeclaration[T]{task, []Task{requirement}})
		case request := <-handler.resolveQueue:
			comms.RequestResolution(request)
		case _, isOpen := <-handler.waitRequests:
			if !isOpen {
				/* This channel being closed is a signal that the task is complete. */
				comms.MarkDone(task)
			} else {
				comms.MarkWaiting(task)
			}
			return
		}
	}
}

func dependencyQueueSize(maxParallelTasks uint) uint {
	return 4 * maxParallelTasks
}

func (manager *taskManager) execute(mainTask Task, maxParallelTasks uint) {
	if maxParallelTasks <= 0 {
		panic("numJobs must be positive!")
	}
	/* No need to close these channels since it wouldn't signal anything anyway. */
	comms := managerComms{
		doneQueue: make(chan *taskEntry, maxParallelTasks),
		/* Queue of requests for tasks to wait. */
		waitQueue: make(chan *taskEntry, maxParallelTasks),
		/* Queue of dependency declarations. */
		dependencyQueue: make(chan dependencyDeclaration[*taskEntry], dependencyQueueSize(maxParallelTasks)),
		resolutionQueue: make(chan resolveRequest, maxParallelTasks),
	}

	manager.run(manager.resolve(mainTask), &comms)

	for manager.numExecuting > 0 {
		select {
		case doneTask := <-comms.doneQueue:
			manager.numExecuting--
			manager.processCompleteTask(doneTask)
		case waitingTask := <-comms.waitQueue:
			manager.numWaiting++
			manager.processWaitingTask(waitingTask)
		case requirement := <-comms.dependencyQueue:
			manager.processRequirement(requirement.dependent, requirement.dependencies)
		case request := <-comms.resolutionQueue:
			/* This will not block with the implementation of chanMessageCallbacks that we have, since
			only one item will ever get placed on the callback channel, and it is a buffered channel. */
			request.callback <- manager.resolve(request.toResolve)
		}

		for manager.numExecuting < maxParallelTasks && !manager.taskQueue.IsEmpty() {
			manager.run(manager.taskQueue.Dequeue())
		}
	}
}
