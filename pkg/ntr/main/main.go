/*
Provides a common main function used when running a named task registerer as the main task.
Includes parsing of flags.
*/
package ntrmain

import (
	"fmt"
	"os"
	"strconv"

	"gitlab.com/kyle_anderson/nbt/pkg/nbt"
	"gitlab.com/kyle_anderson/nbt/pkg/ntr"
)

func Run(registeredTasks map[string]ntr.TaskSupplier, args []string) {
	parsed, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to parse arguments: ", err)
		os.Exit(1)
	}
	task, err := ntr.New(registeredTasks, parsed.namedTasks)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to instantiate named task registerer: ", err)
		os.Exit(1)
	}
	seenErrs := false
	errs := nbt.Start(task, parsed.numJobs)
	for err := range errs {
		seenErrs = true
		fmt.Fprintln(os.Stderr, err)
	}
	if seenErrs {
		os.Exit(1)
	}
}

/* Simple storage for command-line provided arguments. */
type commandLineArgs struct {
	namedTasks []string
	numJobs    uint
}

func parseArgs(args []string) (parsed commandLineArgs, err error) {
	/* Marks the end of all flags. */
	const flagEndDelimiter = "--"
	i := uint(0)
	for isFlag(args[i]) {
		switch args[i] {
		case "-j", "--jobs":
			i++
			if numJobs, conversionErr := strconv.ParseUint(args[i], 10, 32); conversionErr != nil {
				err = fmt.Errorf("failed to parse number of jobs: %w", conversionErr)
				return
			} else {
				parsed.numJobs = uint(numJobs)
			}
			i++
		default:
			err = fmt.Errorf(`unknown flag: %v`, args[i])
			return
		}
	}
	if args[i] == flagEndDelimiter {
		i++
	}
	parsed.namedTasks = args[i:]
	return
}

/*
Determines if the given argument is a flag argument or not.
-- is not considered a flag argument.
*/
func isFlag(arg string) bool {
	return (len(arg) == 2 && arg[0] == '-' && arg[1] != '-') || (len(arg) > 2 && arg[:2] == "--")
}
