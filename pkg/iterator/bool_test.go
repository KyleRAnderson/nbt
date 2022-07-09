package iterator

import (
	"fmt"
	"testing"
)

func TestBoolFunctions(t *testing.T) {
	t.Run(`with bool slice iterators`, func(t *testing.T) {
		for _, isAny := range []bool{false, true} {
			isAny := isAny // Capture
			var fn func(Iterator[bool]) (bool, error)
			var fnName string
			if isAny {
				fnName = "AnyBool"
				fn = AnyBool
			} else {
				fnName = "AllBool"
				fn = AllBool
			}
			t.Run(fmt.Sprintf("with %s function", fnName), func(t *testing.T) {
				for testNo, test := range []struct {
					input   []bool
					allTrue bool
					anyTrue bool
				}{
					{[]bool{}, true, false},
					{[]bool{true}, true, true},
					{[]bool{false, false, true, true}, false, true},
					{make([]bool, 100), false, false},
				} {
					test := test // Capture
					t.Run(fmt.Sprint("case ", testNo), func(t *testing.T) {
						t.Log("test: ", test)
						/* No need to close a slice iterator. */
						receivedAll, err := fn(SliceIterator(test.input))
						expected := (isAny && test.anyTrue) || (!isAny && test.allTrue) // xor
						if err != nil {
							t.Errorf("unexpected error: %#v", err)
						} else if receivedAll != expected {
							t.Errorf("expected: %v, received: %v", expected, receivedAll)
						}
					})
				}
			})
		}
	})
}
