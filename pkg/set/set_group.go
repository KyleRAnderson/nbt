/* Provides an immutable set implementation which is simply a group of sets. */

package set

import "gitlab.com/kyle_anderson/go-utils/pkg/iterator"

type SetGroup[T any] []Set[T]

func (s SetGroup[T]) Contains(elem T) (contained bool) {
	for i := 0; i < len(s) && !contained; i++ {
		contained = s[i].Contains(elem)
	}
	return
}

func (s SetGroup[T]) It() iterator.Iterator[T] {
	return iterator.Flatten[T](
		iterator.Map[Set[T]](iterator.SliceIterator(s), func(s Set[T]) (iterator.Iterator[T], error) {
			return s.It(), nil
		}),
	)
}
