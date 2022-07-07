/* An implementation of set that only assumes comparability for its elements. */

package set

type ComparableSet[T comparable] map[T]struct{}

func NewComparable[T comparable](elements ...T) ComparableSet[T] {
	set := ComparableSet[T](make(map[T]struct{}, len(elements)))
	for _, e := range elements {
		set.Add(e)
	}
	return set
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
