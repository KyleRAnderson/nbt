package main

import (
	"fmt"
	"os"

	"gitlab.com/kyle_anderson/nbt/pkg/generator"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, `expected 1 argument, received %d
Usage:
	nbtgen {path to task file definition directory}
`, len(os.Args))
		os.Exit(1)
	}
	err := generator.Generate(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
