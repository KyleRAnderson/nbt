package iterator

type Iterable[T any] interface {
	It() Iterator[T]
}
