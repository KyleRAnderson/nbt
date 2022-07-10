package set

import (
	"testing"
)

func TestOperations(t *testing.T) {
	items := []int{1, 2, 3, 4}
	/* Use interface type Set so that these tests work for all set implementations. */
	var s Set[int] = NewComparable(items...)

	t.Run("iterator", func(t *testing.T) {
		itemsM := make(map[int]struct{}, len(items))
		for _, value := range items {
			itemsM[value] = struct{}{}
		}
		it := s.It()
	loop:
		for {
			elem, err := it.Next()
			switch {
			case err == nil:
				if _, contained := itemsM[elem]; !contained {
					t.Errorf("received unexpected value: %v", elem)
				} else {
					delete(itemsM, elem)
				}
			case err.IsDone():
				break loop
			default:
				t.Fatalf("unexpected error: %v", err)
			}
		}
		for k := range itemsM {
			t.Errorf("did not receive value %v", k)
		}
	})

	t.Run("contains and remove", func(t *testing.T) {
		/* Reverse the slice, just to mix things up. */
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
		for _, value := range items {
			if !s.Contains(value) {
				t.Errorf("Contains(%v) was false when it should have been true", value)
			}
			s.Remove(value)
			if s.Contains(value) {
				t.Errorf("Contains(%v) was true after removing the item when it should be false", value)
			}
		}
	})

	t.Run("contains and add", func(t *testing.T) {
		toAdd := []int{9, 928, -910, 0b11101}
		for _, value := range toAdd {
			if s.Contains(value) {
				t.Errorf("set contains %v prior to adding it", value)
			}
			s.Add(value)
			if !s.Contains(value) {
				t.Errorf("set does not contain %v after adding it", value)
			}
		}
	})
}
