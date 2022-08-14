package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

/*
Determines if the given task meets the preconditions for being a task that can be named
on the command line.
*/
func meetsNamedTaskPrecondition(task *taskFunc) bool {
	return len(task.Params) == 0 || len(task.Params) == 1 && len(task.Params[0].Names) == 1 && task.Params[0].Type == "string"
}

func createHelperFile(dir string, funcInfos <-chan *taskFunc) error {
	const helperFileBasename = "helper." + generatedFileExt + ".go"
	file, err := os.Create(filepath.Join(dir, helperFileBasename))
	if err != nil {
		return fmt.Errorf(`generator.createHelperFile: failed to open file: %w`, err)
	}
	defer file.Close()
	buf := bufio.NewWriter(file)
	if err := generateHelper(buf, funcInfos); err != nil {
		return err
	}
	buf.Flush()
	if err := reformatFile(file); err != nil {
		return fmt.Errorf(`generator.createHelperFile: failed to autoformat result: %w`, err)
	}
	return nil
}

/*
Generates the helper file given the names of the necessary constants.
The names should be in a deterministic order such that repetitive calls to the
generator always yield the same output.
*/
func generateHelper(writer *bufio.Writer, funcInfos <-chan *taskFunc) error {
	if err := writeGeneratedHeader(writer); err != nil {
		return fmt.Errorf(`generator.generateHelper: failed to add generated message header: %w`, err)
	}
	if _, err := writer.WriteString(`package main
import (
	"gitlab.com/kyle_anderson/nbt/pkg/nbt"
	"gitlab.com/kyle_anderson/nbt/pkg/ntr"
)
`); err != nil {
		return fmt.Errorf(`generator.generateHelper: failed to write top line: %w`, err)
	}
	filteredTasks := make(chan *taskFunc, 2)
	go func() {
		defer close(filteredTasks)
		for fn := range funcInfos {
			if meetsNamedTaskPrecondition(fn) {
				filteredTasks <- fn
			}
		}
	}()
	t := template.Must(template.New("named tasks").Parse(`func getNamedTasks() map[string]ntr.TaskSupplier {
	result := make(map[string]ntr.TaskSupplier)
{{- range . }}
	result[{{ .NameWithoutTask | printf "%q" }}] = func(arg string) (t nbt.Task, err error) { 
{{- if eq (len .Params) 0 }}
		t = {{.ConstructorName}}()
{{- else }}
		t = {{.ConstructorName}}(arg)
{{- end }}
		return
	}
{{- end }}
	return result
}`))
	if err := t.Execute(writer, filteredTasks); err != nil {
		return fmt.Errorf(`generator.generateHelper: failed to execute getNamedTasks template: %w`, err)
	}
	return nil
}
