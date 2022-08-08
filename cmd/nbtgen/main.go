package main

import (
	"fmt"
	"os"

	"gitlab.com/kyle_anderson/nbt/pkg/generator"
)

func main() {
	err := generator.Generate("pkg/generator/test_cases/1")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
