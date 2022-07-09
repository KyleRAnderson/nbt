package iterator

import (
	"errors"
)

type Iterator[T any] interface {
	Next() (T, *IterError)
	/* Close releases any resources that may be in use by the iterator.
	This may do nothing.
	Mutating an iterator (such as by advancing it) after its Close method has been called
	results in undefined behaviour. */
	Close()
}

/* Wrapper around error to provide utility methods which add logic
for detecting the end of the iterator. This avoids publicly exposing the `errDoneIteration` variable,
keeping the implemenetation details within this package. */
type IterError struct {
	error
}

func (err IterError) IsDone() bool {
	return errors.Is(err.error, errDoneIteration)
}

func (err IterError) Unwrap() error {
	return err.error
}

var errDoneIteration = errors.New("no more items in iterator")

func DoneIterationErr() *IterError {
	return &IterError{errDoneIteration}
}
