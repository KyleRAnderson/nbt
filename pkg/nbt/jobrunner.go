package nbt

func runJob[T Task](queue <-chan T, c callbacker[T]) {
	/* Note that we use generics in the declaration so that this function cannot manipulate
	the task entry, while still keeping the type information so that the caller does not have to
	perform type assertions when receiving things on channels. */
	for task := range queue {
		h := handler[T]{task, c}
		task.Perform(&h)
	}
}

type callbacker[T Task] interface {
	OnTaskComplete(T)
	OnTaskWaiting(T)
	OnRequire(dependent T, dependencies []Task)
	Resolve(Task) Task
}
