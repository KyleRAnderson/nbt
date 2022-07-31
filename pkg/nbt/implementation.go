package nbt

import (
	"sync"

	"gitlab.com/kyle_anderson/go-utils/pkg/set"
)

func Start(mainTask Task, numJobs uint) {
	if numJobs == 0 {
		panic("numJobs must be positive!")
	}
	/* Use plain tasks for the job queue to prevent the other goroutines
	from modifying stuff in *taskEntry. */
	taskQueue := make(chan *taskEntry, numJobs)
	defer close(taskQueue)
	/* No need to close these other channels since it wouldn't signal anything anyway. */
	doneQueue := make(chan *taskEntry, numJobs)
	waitQueue := make(chan *taskEntry, numJobs)
	dependencyQueue := make(chan dependencyDeclaration[*taskEntry], numJobs)
	resolutionQueue := make(chan resolveRequest, numJobs)

	wg := &sync.WaitGroup{}
	wg.Add(int(numJobs))
	var callbacks callbacker[*taskEntry] = &chanMessageCallbacks[*taskEntry]{doneQueue, waitQueue, dependencyQueue, resolutionQueue}
	for i := uint(0); i < numJobs; i++ {
		go func() {
			runJob(taskQueue, callbacks)
			wg.Done()
		}()
	}

	manager := newTaskManager(taskQueue)

	select {
	case doneTask := <-doneQueue:
		manager.processCompleteTask(doneTask)
	case waitingTask := <-waitQueue:
		manager.processWaitingTask(waitingTask)
	case requirement := <-dependencyQueue:
		manager.processRequirement(requirement.dependent, requirement.dependencies)
	case request := <-resolutionQueue:
		/* This will not block with the implementation of chanMessageCallbacks that we have, since
		only one item will ever get placed on the callback channel, and it is a buffered channel. */
		request.callback <- manager.resolve(request.toResolve)
	}

	wg.Wait()
}

type chanMessageCallbacks[T Task] struct {
	/* Queue of tasks that have completed. */
	doneQueue chan<- T
	/* Queue of tasks that have requested to wait. */
	waitQueue chan<- T
	/* Queue of new dependencies. */
	dependencyQueue chan<- dependencyDeclaration[T]
	resolveRequests chan<- resolveRequest
}

func (c *chanMessageCallbacks[T]) OnTaskComplete(t T) {
	c.doneQueue <- t
}
func (c *chanMessageCallbacks[T]) OnTaskWaiting(t T) {
	c.waitQueue <- t
}
func (c *chanMessageCallbacks[T]) OnRequire(dependent T, dependencies []Task) {
	go func() {
		c.dependencyQueue <- dependencyDeclaration[T]{dependent, dependencies}
	}()
}
func (c *chanMessageCallbacks[T]) Resolve(t Task) Task {
	resolution := make(chan Task, 1)
	c.resolveRequests <- resolveRequest{resolution, t}
	return <-resolution
}

type dependencyDeclaration[T Task] struct {
	dependent    T
	dependencies []Task
}

type resolveRequest struct {
	callback  chan<- Task
	toResolve Task
}

type taskEntry struct {
	Task
	/* Slice of tasks that are dependent and still waiting on this task. */
	dependents []*taskEntry
	/* Tasks upon which this task depends. */
	dependencies set.Set[*taskEntry]
}

/* Returns true if this task is ready to execute, when all of its dependencies have been met, false otherwise. */
func (te *taskEntry) IsReady() bool {
	return te.dependencies.Size() <= 0
}
