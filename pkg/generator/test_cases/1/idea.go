package main

import (
	"gitlab.com/kyle_anderson/nbt/pkg/nbt"
)

/* Idea: We could generate task type declarations from functions like the following.
The generated code would be placed in the same package as these functions so that
even the linters would work, after generation is done. */

func TaskCompileC(h nbt.Handler, source, dest string) error {
	/* Compile C file */
	return nil
}

func TaskLinkProgram(h nbt.Handler) error {
	h.Require(NewTaskCompileC("hello.c", "hello.o"))
	h.Require(NewTaskCompileC("main.c", "main.o"))
	h.Wait()
	/* Link program. */
	return nil
}
