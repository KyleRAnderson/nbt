/* Common iterators. */

package iterator

import "fmt"

type (
	mapIterator[K comparable, V any] struct {
		Source map[K]V
		items  chan KeyValuePair[K, V]
		/* A channel signal for when the iterator has been closed. */
		closer chan struct{}
	}
	sliceIterator[E any] struct {
		Source []E
		i      int
		NopClose
	}
	chainIterator[E any] struct {
		/* Iterator through the iterator sources. */
		sourceIt Iterator[Iterator[E]]
		/* Current iterator from which items will be grabbed. */
		current Iterator[E]
	}
	mappedIterator[Source, Dest any] struct {
		sourceIt    Iterator[Source]
		transformer func(Source) (Dest, error)
	}
)

func SliceIterator[E any](source []E) *sliceIterator[E] {
	return &sliceIterator[E]{Source: source}
}

func (it *sliceIterator[E]) Next() (elem E, err *IterError) {
	defer func() {
		if r := recover(); r != nil {
			err = DoneIterationErr()
		} else {
			it.i++
		}
	}()
	elem = it.Source[it.i]
	return
}

func MapIterator[K comparable, V any](source map[K]V) *mapIterator[K, V] {
	it := mapIterator[K, V]{Source: source, items: make(chan KeyValuePair[K, V]), closer: make(chan struct{})}
	go func() {
		defer close(it.items)
		for k, v := range it.Source {
			pair := KeyValuePair[K, V]{k, v}
			select {
			case <-it.closer:
				return
			case it.items <- pair:
			}
		}
	}()
	return &it
}

type KeyValuePair[K, V any] struct {
	Key   K
	Value V
}

func (it *mapIterator[K, V]) Next() (elem KeyValuePair[K, V], err *IterError) {
	if item, ok := <-it.items; ok {
		elem = item
	} else {
		err = DoneIterationErr()
	}
	return
}

func (it *mapIterator[K, V]) Close() {
	close(it.closer)
}

func Chain[E any](sources ...Iterator[E]) *chainIterator[E] {
	return Flatten[E](SliceIterator(sources))
}

func Flatten[E any](source Iterator[Iterator[E]]) *chainIterator[E] {
	return &chainIterator[E]{source, nil}
}

func (c *chainIterator[E]) Next() (elem E, err *IterError) {
	var sourceItErr, nestItErr *IterError

	for {
		if c.current == nil {
			c.current, sourceItErr = c.sourceIt.Next()
		}
		if sourceItErr != nil {
			if sourceItErr.IsDone() {
				err = sourceItErr
				break
			} else {
				/* It is expected that sourceIt is a reliable, non error-returning iterator. If it does
				return an error that isn't simply an indication of being done, then something is wrong. */
				panic("iterator: chain iterator sourceIt.Next() errored before completion: " + err.Error())
			}
		}
		elem, nestItErr = c.current.Next()
		if nestItErr != nil {
			if nestItErr.IsDone() {
				c.current.Close()
				c.current = nil
				continue
			} else {
				err = &IterError{fmt.Errorf(`chainIterator: one of chained iterators returned error: %w`, nestItErr)}
			}
		}
		break
	}
	return
}

func (c *chainIterator[E]) Close() {
	if c.current != nil {
		c.current.Close()
	}
	for {
		elem, err := c.sourceIt.Next()
		/* In the case that err != nil && !err.IsDone(), we still need to try to continue through all the
		iterators and close them. */
		switch {
		case err == nil:
			elem.Close()
		case err.IsDone():
			return
		}
	}
}

/* Creates a new iterator which transforms the elements of the source iterator prior to being returned.
This iterator assumes complete control over the source iterator, including over `Close()`. */
func Map[Source, Dest any](source Iterator[Source], transformer func(Source) (Dest, error)) *mappedIterator[Source, Dest] {
	return &mappedIterator[Source, Dest]{source, transformer}
}

func (m *mappedIterator[Source, Dest]) Next() (elem Dest, err *IterError) {
	var sourceElem Source
	sourceElem, err = m.sourceIt.Next()
	if err != nil {
		if !err.IsDone() {
			err = &IterError{fmt.Errorf(`mappedIterator: source iterator errored when retrieving next element: %w`, err)}
		}
		return
	}
	elem, transformErr := m.transformer(sourceElem)
	if transformErr != nil {
		err = &IterError{fmt.Errorf(`mappedIterator: transformer errored on input %v: %w`, sourceElem, transformErr)}
	}
	return
}

func (m *mappedIterator[Source, Dest]) Close() { m.sourceIt.Close() }
