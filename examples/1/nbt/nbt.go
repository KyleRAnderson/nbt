package main

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"os/exec"

	"gitlab.com/kyle_anderson/nbt/pkg/nbt"
)

const (
	hashBaseCompileC uint64 = iota
	hashBaseLinkProgram
)

type taskCompileC struct {
	source, dest string
}

func (t *taskCompileC) Hash() uint64 {
	h := fnv.New64()
	if err := binary.Write(h, binary.LittleEndian, hashBaseCompileC); err != nil {
		panic(fmt.Errorf(`(*taskCompileC).Hash: error writing hash: %w`, err))
	}
	h.Write([]byte(t.source))
	h.Write([]byte(t.dest))
	return h.Sum64()
}
func (t *taskCompileC) Matches(other nbt.Task) bool {
	if tcc, ok := other.(*taskCompileC); ok {
		return tcc.dest == t.dest
	}
	return false
}

func (t *taskCompileC) Perform(h nbt.Handler) error {
	stdout, err := exec.Command("gcc", "-o", t.dest, "-c", t.source).CombinedOutput()
	fmt.Println(string(stdout), err)
	return err
}

type taskLinkProgram struct{}

func (t *taskLinkProgram) Hash() uint64 {
	h := fnv.New64()
	if err := binary.Write(h, binary.LittleEndian, hashBaseLinkProgram); err != nil {
		panic(fmt.Errorf(`(*taskLinkProgram).Hash: error writing hash: %w`, err))
	}
	return h.Sum64()
}
func (t *taskLinkProgram) Matches(other nbt.Task) bool {
	_, ok := other.(*taskLinkProgram)
	return ok
}

func (t *taskLinkProgram) Perform(h nbt.Handler) error {
	h.Require(&taskCompileC{"hello.c", "hello.o"})
	h.Require(&taskCompileC{"main.c", "main.o"})
	h.Wait()
	stdout, err := exec.Command("gcc", "-o", "hello.out", "hello.o", "main.o").CombinedOutput()
	fmt.Println(string(stdout), err)
	return err
}

func main() {
	/* For the completed software, an automatic main task would be created
	which would read os.Args and find the tasks listed there, then require them and wait. */
	nbt.Start(&taskLinkProgram{}, 1)
}
