package main

import (
	"fmt"
	"os"

	"gitlab.com/kyle_anderson/nbt/pkg/generator"
)

func main() {
	var workingDirectory string
	switch len(os.Args) {
	case 1:
		workingDirectory = "."
	case 2:
		workingDirectory = os.Args[1]
	default:
		fmt.Fprintf(os.Stderr, `expected 1 argument, received %d
	Usage:
	nbtgen [path to working directory]
	`, len(os.Args))
		os.Exit(1)
	}
	err := generator.Generate(workingDirectory)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
