package nbt

import "gitlab.com/kyle_anderson/go-utils/pkg/set"

func newTaskManager(taskQueue chan<- *taskEntry) *taskManager {
	return &taskManager{make(map[int][]*taskEntry), taskQueue}
}

type taskManager struct {
	registry  map[int][]*taskEntry
	taskQueue chan<- *taskEntry
}

func (tm *taskManager) processCompleteTask(task *taskEntry) {
	for _, dependent := range task.dependents {
		dependent.dependencies.Remove(task)
		tm.processWaitingTask(dependent)
	}
}

func (tm *taskManager) processWaitingTask(task *taskEntry) {
	if task.IsReady() {
		tm.taskQueue <- task
	}
}

func (tm *taskManager) processRequirement(dependent *taskEntry, dependencies []Task) {
	for _, dependency := range dependencies {
		resolvedDependency := tm.resolve(dependency)
		dependent.dependencies.Add(resolvedDependency)
		resolvedDependency.dependents = append(resolvedDependency.dependents, dependent)
		/* Tasks that have not yet executed will have empty dependencies, so should be ready. */
		tm.processWaitingTask(resolvedDependency)
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
		currentInstance = &taskEntry{task, make([]*taskEntry, 0), set.NewComparable[*taskEntry]()}
		taskChain = append(taskChain, currentInstance)
	}
	return
}
