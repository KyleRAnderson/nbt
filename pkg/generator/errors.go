package generator

import (
	"errors"
	"fmt"
)

/* Errors outputted by the generator. */

type ErrParse struct{ nested error }

func (e *ErrParse) Error() string {
	return fmt.Sprint(`generator: failed to parse source files: %w`, e.nested)
}
func (e *ErrParse) Unwrap() error {
	return e.nested
}

func ErrNoMainPackage() error {
	return errors.New(`generator: no main package found in input directory`)
}

type ErrFileProcessing struct {
	embedded              error
	inputFile, outputFile string
}

func (e *ErrFileProcessing) Unwrap() error { return e.embedded }
func (e *ErrFileProcessing) Error() string {
	return fmt.Sprintf(`failed to process input file %q to %q, err: %v`, e.inputFile, e.outputFile, e.embedded)
}
