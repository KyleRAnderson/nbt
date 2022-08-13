package ntr

import (
	"errors"
	"fmt"
	"testing"

	"gitlab.com/kyle_anderson/go-utils/pkg/set"
	"gitlab.com/kyle_anderson/nbt/pkg/nbt"
)

type mockTask uint

func (mockTask) Hash() uint64 { return 0 }
func (m mockTask) Matches(other nbt.Task) bool {
	if converted, ok := other.(mockTask); ok {
		return converted == m
	}
	return false
}
func (mockTask) Perform(nbt.Handler) error { return nil }

type monitoredHandler struct {
	onRequire func(nbt.Task)
}

func (monitoredHandler) Wait()                     {}
func (monitoredHandler) Resolve(nbt.Task) nbt.Task { return nil }
func (m *monitoredHandler) Require(t nbt.Task)     { m.onRequire(t) }

func TestNamedTaskRequirer(t *testing.T) {
	t.Run(`basic checks`, func(t *testing.T) {
		registeredTasks := map[string]TaskSupplier{
			"one":   func(s string) (nbt.Task, error) { return mockTask(1), nil },
			"two":   func(s string) (nbt.Task, error) { return mockTask(2), nil },
			"three": func(s string) (nbt.Task, error) { return mockTask(3), nil },
		}
		for i, test := range []struct {
			input    []string
			expected set.Set[mockTask]
		}{
			{[]string{"one", "two", "three"}, set.NewComparable[mockTask](1, 2, 3)},
			{[]string{}, set.NewComparable[mockTask]()},
			{[]string{"two"}, set.NewComparable[mockTask](2)},
			{[]string{"three", "two"}, set.NewComparable[mockTask](2, 3)},
			{[]string{"one"}, set.NewComparable[mockTask](1)},
		} {
			test := test // Capture
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				nt, err := New(registeredTasks, test.input)
				if err != nil {
					t.Error(`New: received unexpected error: `, err)
				} else {
					var requiredTasks []mockTask
					handler := monitoredHandler{func(t nbt.Task) { requiredTasks = append(requiredTasks, t.(mockTask)) }}
					nt.Perform(&handler)
					for _, task := range requiredTasks {
						if !test.expected.Contains(task) {
							t.Errorf("unexpected task requirement: %v", task)
						} else {
							test.expected.Remove(task)
						}
					}
					it := test.expected.It()
				loop:
					for {
						task, err := it.Next()
						switch {
						case err.IsDone():
							break loop
						case err.Unwrap() == nil:
							t.Errorf(`task %v unexecuted`, task)
						default:
							t.Fatal(`unexpected error while iterating expected tasks: `, err.Unwrap())
						}
					}
				}
			})
		}
	})

	t.Run(`with task construction errors`, func(t *testing.T) {
		oneErr := errors.New(`one`)
		_, err := New(map[string]TaskSupplier{
			"one": func(s string) (nbt.Task, error) { return nil, oneErr },
		}, []string{"one"})
		if err == nil {
			t.Error(`expected error but did not receive one`)
		} else {
			if !errors.Is(err, oneErr) {
				t.Errorf(`expected error to be oneErr, but got: %#v`, err)
			}
			var constructionErr *ErrTaskConstruction
			if !errors.As(err, &constructionErr) {
				t.Errorf(`unexpected error type: %T`, err)
			}
		}
	})

	t.Run(`with a nonexistent task`, func(t *testing.T) {
		_, err := New(map[string]TaskSupplier{
			"one": func(string) (nbt.Task, error) { return mockTask(1), nil },
			"two": func(string) (nbt.Task, error) { return mockTask(2), nil },
		}, []string{"one", "nonexistent", "two"})
		if err == nil {
			t.Error(`expected error but did not receive one`)
		} else {
			var receivedErr *ErrTaskNotFound
			if !errors.As(err, &receivedErr) {
				t.Errorf(`unexpected error type: %#v`, err)
			} else if *receivedErr != (ErrTaskNotFound{"nonexistent"}) {
				t.Errorf(`unexpected error value: %#v`, receivedErr)
			}
		}
	})
}
