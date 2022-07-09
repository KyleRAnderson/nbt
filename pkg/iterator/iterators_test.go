package iterator

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

func generateRandomIntSlice(r *rand.Rand, length uint) []int {
	result := make([]int, length)
	for i := range result {
		result[i] = r.Int()
	}
	return result
}

func generateRandomIntMap(r *rand.Rand, length uint) map[int]int {
	result := make(map[int]int, length)
	for n := uint(0); n < length; n++ {
		for {
			next := r.Int()
			if _, alreadyContained := result[next]; !alreadyContained {
				result[next] = r.Int()
				break
			}
		}
	}
	return result
}

func checkIteratorWithSlice[E comparable](t *testing.T, expectedElements []E, it Iterator[E]) {
	for i, expectedElem := range expectedElements {
		receivedElem, err := it.Next()
		if err != nil {
			t.Errorf(`received error from slice iterator: %v`, err)
		}
		if receivedElem != expectedElem {
			t.Errorf(`index: %d, expected: %v, received: %v`, i, expectedElem, receivedElem)
		}
	}
	_, err := it.Next()
	if !err.IsDone() {
		t.Errorf(`expected iterator to be done, but was not`)
	}
}

func TestSliceIterator(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(3))
	for testNo, test := range [][]int{
		{},
		{1},
		{1, 2, 3, 4, 5},
		{6, 9, -385, 14543, 0b100101000101, -0x83},
		generateRandomIntSlice(r, 43),
		generateRandomIntSlice(r, 1_085),
	} {
		test := test // Capture
		t.Run(fmt.Sprint(`case`, testNo), func(t *testing.T) {
			t.Log(`input:`, test)
			it := SliceIterator(test)
			defer it.Close()
			checkIteratorWithSlice[int](t, test, it)
		})
	}
}

func copyMap[K comparable, V any](in map[K]V) map[K]V {
	copy := make(map[K]V, len(in))
	for k, v := range in {
		copy[k] = v
	}
	return copy
}

func TestMapIterator(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(-5190))
	for testNo, test := range []map[int]int{
		nil,
		{},
		{0: 1, 2: 3, 4: 5},
		{-839: 45501, 8902: 8240, 999: 888, 0b1010100011: 0o77, -1: 0},
		generateRandomIntMap(r, 5),
		generateRandomIntMap(r, 42),
		generateRandomIntMap(r, 1_085),
	} {
		test := test // Capture
		t.Run(fmt.Sprint(`case `, testNo), func(t *testing.T) {
			checkReceivedElem := func(elem KeyValuePair[int, int]) {
				expectedValue, shouldExpectValue := test[elem.Key]
				if !shouldExpectValue {
					t.Errorf(`received unexpected key: %v`, elem.Key)
				} else if expectedValue != elem.Value {
					t.Errorf(`value for key %v did not match, expected: %v, received: %v`, elem.Key, expectedValue, elem.Value)
				}
				delete(test, elem.Key)
			}

			it := MapIterator(copyMap(test))
			defer it.Close()
		loop:
			for {
				receivedElem, err := it.Next()
				switch {
				case err == nil:
					checkReceivedElem(receivedElem)
				case err.IsDone():
					break loop
				default:
					t.Errorf(`received unexpected error: %v`, err)
				}
			}
			for k, v := range test {
				t.Errorf(`never received (key, value) pair: (%v, %v)`, k, v)
			}
		})
	}
}

func between(r *rand.Rand, min, max int) int {
	return rand.Intn(max-min+1) + min
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

/* Randomly splits the input slice `numSplits` times, reslting in `numSplits + 1` slices.
Panics if such a split is impossible. */
func randomSplice[E any](r *rand.Rand, source []E, numSplits int) [][]E {
	if numSplits >= len(source) {
		panic("randomSplice: number of splits is too large for the length of the input slice")
	} else if numSplits < 0 {
		panic("randomSplice: numSplits is negative")
	}
	if numSplits == 0 {
		return [][]E{source}
	}
	splitPoint := between(r, 1, Max(len(source)-1, 1))
	numSplits--
	leftSplit, rightSplit := source[:splitPoint], source[splitPoint:]
	/* Let nl, nr be  the number of left and right splits, respectively.
	Let n be the number of splits (`numSplits`)
	Let ll, lr be len(leftSplit) and len(rightSplit) respectively.
	1) nl + nr = n
	2) 0 <= nl < ll
	3) 0 <= nr < lr

	4) From (1), nr = n - nl
	5) From (4) and (3), n - nl < lr
	6) From (5), n - lr < nl
	7) From (1), nl <= n
	8) Conclusion: max(0, n - lr + 1) <= nl <= min(n, ll - 1)
	*/
	numLeftSplits := between(r, Max(0, numSplits-len(rightSplit)+1), Min(numSplits, len(leftSplit)-1))
	return append(randomSplice(r, leftSplit, numLeftSplits), randomSplice(r, rightSplit, numSplits-numLeftSplits)...)
}

type closeWrapper[E any] struct {
	Iterator[E]
	customClose func(Iterator[E])
}

func (cw *closeWrapper[E]) Close() {
	cw.customClose(cw.Iterator)
}
func addCustomClose[E any](it Iterator[E], newClose func(Iterator[E])) Iterator[E] {
	return &closeWrapper[E]{it, newClose}
}

func TestChainIterator(t *testing.T) {
	t.Parallel()
	t.Run(`with empty iterators`, func(t *testing.T) {
		for _, numIterators := range []uint{0, 1, 2, 3, 5, 10, 0x100} {
			numIterators := numIterators // Capture
			t.Run(fmt.Sprintf(`%d empty iterators`, numIterators), func(t *testing.T) {
				for _, test := range []struct {
					name string
					test func(*testing.T, Iterator[int])
				}{
					{"consuming all iterators", func(t *testing.T, it Iterator[int]) {
						t.Cleanup(func() { it.Close() })
						_, err := it.Next()
						if err == nil {
							t.Errorf("nil error")
						} else if !err.IsDone() {
							t.Errorf(`expected iterator to be instantly consumed, but was not`)
						}
					}},
					{"closing without consumption", func(t *testing.T, it Iterator[int]) {
						it.Close()
					}},
				} {
					test := test // Capture
					t.Run(test.name, func(t *testing.T) {
						t.Parallel()
						allIt := make([]Iterator[int], numIterators)
						closed := make([]bool, numIterators)
						/* Use a new iterator each time for a better test. */
						for i := range allIt {
							i := i // Capture
							allIt[i] = addCustomClose[int](SliceIterator([]int{}), func(it Iterator[int]) { closed[i] = true; it.Close() })
						}
						it := Chain(allIt...)
						test.test(t, it)
						for _, isClosed := range closed {
							if !isClosed {
								t.Errorf("not all iterators were closed: %#v", closed)
								break
							}
						}
					})
				}
			})
		}
	})

	t.Run("with nonempty iterators", func(t *testing.T) {
		r := rand.New(rand.NewSource(9218))
		for testNo, test := range [][]int{
			{1},
			{1, 2},
			{7, -3, 490, 8801, 3701, 0x89},
			generateRandomIntSlice(r, 8),
			generateRandomIntSlice(r, 42),
			generateRandomIntSlice(r, 128),
			generateRandomIntSlice(r, 1049),
		} {
			t.Run(fmt.Sprint("testNo ", testNo), func(t *testing.T) {
				/* Log2 is chosen here just to use a slow growing function. */
				maxNumSplits := int(math.Log2(float64(len(test))))
				spliced := randomSplice(r, test, (maxNumSplits))
				t.Logf("spliced test case: %v", spliced)
				iterators := make([]Iterator[int], len(spliced))
				closed := make([]bool, len(spliced))
				for i := range spliced {
					i := i // Capture
					iterators[i] = addCustomClose[int](SliceIterator(spliced[i]), func(it Iterator[int]) { closed[i] = true; it.Close() })
				}
				chainIt := Chain(iterators...)
				defer chainIt.Close()
				for i, expectedElem := range test {
					receivedElem, err := chainIt.Next()
					if err != nil {
						t.Errorf(`expected no error at element %d but received %v`, i, err)
					} else if expectedElem != receivedElem {
						t.Errorf(`element mismatch at index %d, expected: %v, received: %v`, i, expectedElem, receivedElem)
					}
				}
				unexpectedElem, err := chainIt.Next()
				if err == nil || !err.IsDone() {
					t.Error(`expected chain iterator to be exhausted, but was not, and received `, unexpectedElem)
				}
				for _, isClosed := range closed {
					if !isClosed {
						t.Errorf("not all iterators were closed: %#v", closed)
						break
					}
				}
			})
		}
	})
}
