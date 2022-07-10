/* An implementation of set that only assumes comparability for its elements. */

package set

import "gitlab.com/kyle_anderson/go-utils/pkg/iterator"

type ComparableSet[T comparable] map[T]struct{}

func NewComparable[T comparable](elements ...T) ComparableSet[T] {
	set := ComparableSet[T](make(map[T]struct{}, len(elements)))
	for _, e := range elements {
		set.Add(e)
	}
	return set
}

func AllocateComparable[T comparable](size uint) ComparableSet[T] {
	return make(ComparableSet[T], size)
}

func (s ComparableSet[T]) Contains(element T) bool {
	_, contained := s[element]
	return contained
}

func (s ComparableSet[T]) Add(element T) {
	s[element] = struct{}{}
}

func (s ComparableSet[T]) Remove(element T) {
	delete(s, element)
}

func (s ComparableSet[T]) Size() uint {
	return uint(len(s))
}

func (s ComparableSet[T]) It() iterator.Iterator[T] {
	return iterator.Map[iterator.KeyValuePair[T, struct{}]](iterator.MapIterator(s), func(kvp iterator.KeyValuePair[T, struct{}]) (T, error) {
		return kvp.Key, nil
	})
}
