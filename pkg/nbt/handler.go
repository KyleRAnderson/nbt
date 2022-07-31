package nbt

type handler[T Task] struct {
	task      T
	callbacks callbacker[T]
}

func (h *handler[T]) Require(task Task) {
	h.callbacks.OnRequire(h.task, []Task{task})
}

func (h *handler[T]) Wait() {
	h.callbacks.OnTaskWaiting(h.task)
}

func (h *handler[T]) Resolve(task Task) Task {
	return h.callbacks.Resolve(task)
}
