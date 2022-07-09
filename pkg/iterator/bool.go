package iterator

import (
	"fmt"
)

func AllBool(it Iterator[bool]) (bool, error) {
loop:
	for {
		elem, err := it.Next()
		switch {
		case err == nil:
			if !elem {
				return false, nil
			}
		case err.IsDone():
			break loop
		default:
			return false, fmt.Errorf(`iterator errored: %w`, err)
		}
	}
	return true, nil
}

func AnyBool(it Iterator[bool]) (bool, error) {
	all, err := All(it, func(value bool) bool { return !value })
	return !all, err
}

func predicateNilErr[E any](predicate func(E) bool) func(E) (bool, error) {
	return func(e E) (bool, error) { return predicate(e), nil }
}

func All[E any](it Iterator[E], predicate func(E) bool) (bool, error) {
	return AllBool(Map(it, predicateNilErr(predicate)))
}

func Any[E any](it Iterator[E], predicate func(E) bool) (bool, error) {
	return AnyBool(Map(it, predicateNilErr(predicate)))
}
