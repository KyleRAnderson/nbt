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
