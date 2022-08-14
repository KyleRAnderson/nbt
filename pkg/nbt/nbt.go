package nbt

type Task interface {
	/* Returns a hash for this task that can be used to use it in a map. */
	Hash() uint64
	/* Returns true if this task matches the given task, false otherwise. */
	Matches(Task) bool
	Perform(h Handler) error
}

type Handler interface {
	Require(Task)
	Wait()
	/* Gets the instance of t that has or will actually execute.
	This operation is semi-expensive since the main goroutine must perform the resolution. */
	Resolve(t Task) Task
}

/*
Starts executing the main task, processing all dependencies until completion.
The caller must consume errors outputted in the returned channel, else the
task processing will hang.
*/
func Start(mainTask Task, maxParallelTasks uint) <-chan error {
	return newTaskManager().execute(mainTask, maxParallelTasks)
}
