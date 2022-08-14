package generator

import (
	"bufio"
	"fmt"
	"go/ast"
	"io"
	"os"
	"regexp"
	"text/template"

	"golang.org/x/tools/imports"
)

var extensionReplacer = regexp.MustCompile(`\.go$`)

func mapSrcNameToDestName(sourceFileName string) (destFileName string) {
	return extensionReplacer.ReplaceAllString(sourceFileName, "."+generatedFileExt+".go")
}

type fileProcessingJob struct {
	file     *ast.File
	filename string
}

/*
	Processes incoming file jobs on the given channel.

Any errors encountered in the process will be sent on the errs channel.
This channel will not be closed by the processor so that it can be used
by multiple processors running in parallel.
*/
func processor(jobs <-chan fileProcessingJob, errs chan<- *ErrFileProcessing, taskFuncs chan<- *taskFunc) {
	ccg := chanFuncGatherer{taskFuncs}
	for job := range jobs {
		if err := processFile(&ccg, job); err != nil {
			errs <- err
		}
	}
}

var funcDeclMatcher = regexp.MustCompile(`^Task`)

func nameMatchesTask(decl *ast.FuncDecl) bool {
	return funcDeclMatcher.MatchString(decl.Name.Name)
}
func extractFunctionInformation(fn *ast.FuncDecl) (info *taskFunc) {
	info = &taskFunc{Name: fn.Name.Name}
	if len(fn.Type.Params.List) > 1 {
		inParams := fn.Type.Params.List[1:]
		info.Params = make([]taskParam, len(inParams))
		/* Skip the first parameter, since that's just the handler. */
		for j, inParam := range inParams {
			outParam := &info.Params[j] // reference for convenience
			outParam.Names = make([]string, len(inParam.Names))
			for i, name := range inParam.Names {
				outParam.Names[i] = name.Name
			}
			switch v := inParam.Type.(type) {
			case *ast.Ident:
				outParam.Type = v.Name
			case *ast.SelectorExpr:
				pkg := v.X.(*ast.Ident).Name
				outParam.Type = pkg + v.Sel.Name
			default:
				panic(fmt.Errorf("unhandled parameter type: %T", v))
			}
		}
	}
	return
}

func writeGeneratedHeader(out io.Writer) error {
	_, err := out.Write([]byte("// Code generated by nbtgen. DO NOT EDIT.\n\n"))
	return err
}

func writeHeader(out io.Writer, imports []*ast.ImportSpec) error {
	if err := writeGeneratedHeader(out); err != nil {
		return fmt.Errorf(`generator.writeHeader: failed to write generation comment: %w`, err)
	}
	t := template.Must(template.New(`header`).Parse(`package main
import (
	{{- range . }}
	{{ if .Name }} {{ .Name.Name }} {{ end }} {{ .Path.Value }}
	{{- end }}
)
`))
	if err := t.Execute(out, imports); err != nil {
		return fmt.Errorf(`generator.writeHeader: failed to execute template: %w`, err)
	}
	return nil
}

type chanFuncGatherer struct {
	out chan<- *taskFunc
}

func (ccg *chanFuncGatherer) AddFunc(fn *taskFunc) {
	ccg.out <- fn
}

/* Gathers the list of constants required for the tasks in a file. */
type funcGatherer interface {
	AddFunc(fn *taskFunc)
}

func processFile(fnGath funcGatherer, job fileProcessingJob) *ErrFileProcessing {
	file := job.file
	var output *os.File
	var err error
	outputPath := mapSrcNameToDestName(job.filename)
	formErr := func(e error) *ErrFileProcessing {
		return &ErrFileProcessing{e, job.filename, outputPath}
	}
	if output, err = os.Create(outputPath); err != nil {
		return formErr(fmt.Errorf(`generator.processFile: failed to create output file: %w`, err))
	}
	defer output.Close()
	w := bufio.NewWriter(output)
	if err := writeHeader(w, file.Imports); err != nil {
		return formErr(fmt.Errorf(`generator.processFile: failed to write header: %w`, err))
	}
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && nameMatchesTask(funcDecl) {
			funcInfo := extractFunctionInformation(funcDecl)
			fnGath.AddFunc(funcInfo)
			if err := GenerateTaskType(w, funcInfo); err != nil {
				return formErr(fmt.Errorf(`generator.processFile: failed to generate task type: %w`, err))
			}
		}
	}
	w.Flush()
	if err := reformatFile(output); err != nil {
		return formErr(err)
	}
	return nil
}

func reformatFile(file *os.File) error {
	if rewritten, err := imports.Process(file.Name(), nil, nil); err != nil {
		return fmt.Errorf(`generator.processFile: failed to autoformat: %w`, err)
	} else {
		/* Write the reformatted portion to the same file, and then truncate anything that remains. */
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf(`generator.processFile: failed to seek to beginning prior to rewrite: %w`, err)
		}
		if _, err := file.Write(rewritten); err != nil {
			return fmt.Errorf(`generator.processFile: failed to rewrite with formatted version: %w`, err)
		}
		if currentOffset, err := file.Seek(0, io.SeekCurrent); err == nil {
			file.Truncate(currentOffset)
		} else {
			return fmt.Errorf(`generator.processFile: failed to determine current offset during rewrite: %w`, err)
		}
	}
	return nil
}
