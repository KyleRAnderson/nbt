package set

import "gitlab.com/kyle_anderson/go-utils/pkg/iterator"

type ImmutableSet[T any] interface {
	/* Sets should not return errors while being iterated through. */
	iterator.Iterable[T]
	Contains(T) bool
}

type Set[T any] interface {
	Add(T)
	Remove(T)
	Size() uint
	ImmutableSet[T]
}

/* Performs a set difference, placing the items of the resultant set in `out`. */
func Difference[T any](out, s1, s2 Set[T]) (out2 Set[T]) {
	out2 = out
	iter := s1.It()
	defer iter.Close()
loop:
	for {
		elem, err := iter.Next()
		switch {
		case err == nil:
		case err.IsDone():
			break loop
		default:
			/* It is not expected for sets to return real errors during iteration. */
			panic(err)
		}
		if !s2.Contains(elem) {
			out.Add(elem)
		}
	}
	return
}

func Union[T any](out Set[T], sets ...Set[T]) (out2 Set[T]) {
	out2 = out
	for _, s := range sets {
		func() {
			iter := s.It()
			defer iter.Close()
		loop:
			for {
				elem, err := iter.Next()
				switch {
				case err == nil:
				case err.IsDone():
					break loop
				default:
					/* It is not expected for sets to return real errors during iteration. */
					panic(err)
				}
				out.Add(elem)
			}
		}()
	}
	return
}

func Intersect[T any](out Set[T], sets ...Set[T]) {

}

/* Returns true if s1 is a subset of s2. */
func IsSubset[T any](s1, s2 Set[T]) (isSubset bool) {
	isSubset = true
	iter := s1.It()
	defer iter.Close()
loop:
	for isSubset {
		elem, err := iter.Next()
		switch {
		case err == nil:
		case err.IsDone():
			break loop
		default:
			/* It is not expected for sets to return real errors during iteration. */
			panic(err)
		}
		if !s2.Contains(elem) {
			isSubset = false
		}
	}
	return
}

func Equals[T comparable](s1, s2 Set[T]) bool {
	return IsSubset(s1, s2) && IsSubset(s2, s1)
}
