package iterator

import "errors"

type Iterator[T any] interface {
	Next() (T, IterError)
}

/* Wrapper around error to provide utility methods which add logic
for detecting the end of the iterator. This avoids publicly exposing the `errDoneIteration` variable,
keeping the implemenetation details within this package. */
type IterError struct {
	error
}

func (err *IterError) IsDone() bool {
	return errors.Is(err.error, errDoneIteration)
}

var errDoneIteration = errors.New("no more items in iterator")

func DoneIterationErr() IterError {
	return IterError{errDoneIteration}
}
