/*
ntr: Named task requirer.
Provides a task which accepts tasks registered under names and executes the named tasks.
Useful for parsing tasks from command-line arguments.
*/
package ntr

import (
	"regexp"

	"gitlab.com/kyle_anderson/nbt/pkg/nbt"
)

type TaskSupplier func(string) (nbt.Task, error)

type task struct {
	toPerform []nbt.Task
}

func (t *task) Matches(other nbt.Task) bool { return false }
func (t *task) Hash() uint64                { return 0 }
func (t *task) Perform(h nbt.Handler) error {
	for _, task := range t.toPerform {
		h.Require(task)
	}
	return nil
}

var taskNameRegex = regexp.MustCompile(`^(?P<name>\w+)(?:\((?P<arg>.+)\))?$`)

/*
Creates a new named task requirer, using the given initial registry of tasks.
*/
func New(registeredTasks map[string]TaskSupplier, namedTasks []string) (nbt.Task, error) {
	var t task
	for _, namedTask := range namedTasks {
		matches := taskNameRegex.FindStringSubmatch(namedTask)
		taskName, arg := matches[taskNameRegex.SubexpIndex("name")], matches[taskNameRegex.SubexpIndex("arg")]
		if supplier, ok := registeredTasks[taskName]; ok {
			if task, err := supplier(arg); err == nil {
				t.toPerform = append(t.toPerform, task)
			} else {
				return nil, &ErrTaskConstruction{taskName, arg, err}
			}
		} else {
			return nil, &ErrTaskNotFound{taskName}
		}
	}
	return &t, nil
}
