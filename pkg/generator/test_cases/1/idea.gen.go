package main

import (
	"gitlab.com/kyle_anderson/nbt/pkg/nbt"
)

type TaskCompileCS struct {
	source, dest string
}

func (t *TaskCompileCS) Perform(h nbt.Handler) {
	TaskCompileC(
		h,
		t.source,
		t.dest,
	)
}

type TaskLinkProgramS struct{}

func (t *TaskLinkProgramS) Perform(h nbt.Handler) {
	TaskLinkProgram(
		h,
	)
}
