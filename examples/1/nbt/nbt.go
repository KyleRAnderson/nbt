package main

import (
	"fmt"
	"os/exec"

	"gitlab.com/kyle_anderson/nbt/pkg/nbt"
)

type taskCompileC struct {
	source, dest string
}

func (t *taskCompileC) Hash() uintptr { return 1 /* TODO */ }
func (t *taskCompileC) Matches(other nbt.Task) bool {
	if tcc, ok := other.(*taskCompileC); ok {
		return tcc.dest == t.dest
	}
	return false
}

func (t *taskCompileC) Perform(h nbt.Handler) {
	stdout, err := exec.Command("gcc", "-o", t.dest, "-c", t.source).CombinedOutput()
	fmt.Println(string(stdout), err)
}

type taskLinkProgram struct{}

func (t *taskLinkProgram) Hash() uintptr { return 2 /* TODO */ }
func (t *taskLinkProgram) Matches(other nbt.Task) bool {
	_, ok := other.(*taskLinkProgram)
	return ok
}

func (t *taskLinkProgram) Perform(h nbt.Handler) {
	h.Require(&taskCompileC{"hello.c", "hello.o"})
	h.Require(&taskCompileC{"main.c", "main.o"})
	h.Wait()
	stdout, err := exec.Command("gcc", "-o", "hello.out", "hello.o", "main.o").CombinedOutput()
	fmt.Println(string(stdout), err)
}

func main() {
	/* For the completed software, an automatic main task would be created
	which would read os.Args and find the tasks listed there, then require them and wait. */
	nbt.Start(&taskLinkProgram{}, 2)
}
