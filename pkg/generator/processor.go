package generator

import (
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
func processor(jobs <-chan fileProcessingJob, errs chan<- *ErrFileProcessing) {
	for job := range jobs {
		if err := processFile(job); err != nil {
			errs <- err
		}
	}
}

var funcDeclMatcher = regexp.MustCompile(`^Task`)

func nameMatchesTask(decl *ast.FuncDecl) bool {
	return funcDeclMatcher.MatchString(decl.Name.Name)
}
func extractFunctionInformation(fn *ast.FuncDecl) (info *taskFunc) {
	info = &taskFunc{Name: fn.Name.Name, Params: make([]taskParam, len(fn.Type.Params.List))}
	if len(fn.Type.Params.List) > 1 {
		/* Skip the first parameter, since that's just the handler. */
		for j, inParam := range fn.Type.Params.List[1:] {
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

func writeHeader(out io.Writer, imports []*ast.ImportSpec) error {
	t := template.Must(template.New(`header`).Parse(`package main
import (
	{{ range . }}
	{{ if .Name }} {{ .Name.Name }} {{ end }} {{ .Path.Value }}
	{{ end }}
)
`))
	if err := t.Execute(out, imports); err != nil {
		return fmt.Errorf(`generator.writeHeader: failed to execute template: %w`, err)
	}
	return nil
}

func processFile(job fileProcessingJob) *ErrFileProcessing {
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

	if err := writeHeader(output, file.Imports); err != nil {
		return formErr(fmt.Errorf(`generator.processFile: failed to write header: %w`, err))
	}
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && nameMatchesTask(funcDecl) {
			funcInfo := extractFunctionInformation(funcDecl)
			if err := GenerateTaskType(output, funcInfo); err != nil {
				return formErr(fmt.Errorf(`generator.processFile: failed to generate task type: %w`, err))
			}
		}
	}
	if rewritten, err := imports.Process(output.Name(), nil, nil); err != nil {
		return formErr(fmt.Errorf(`generator.processFile: failed to autoformat: %w`, err))
	} else {
		if _, err := output.Seek(0, io.SeekStart); err != nil {
			return formErr(fmt.Errorf(`generator.processFile: failed to seek to beginning prior to rewrite: %w`, err))
		}
		if _, err := output.Write(rewritten); err != nil {
			return formErr(fmt.Errorf(`generator.processFile: failed to rewrite with formatted version: %w`, err))
		}
		if currentOffset, err := output.Seek(0, io.SeekCurrent); err == nil {
			output.Truncate(currentOffset)
		} else {
			return formErr(fmt.Errorf(`generator.processFile: failed to determine current offset during rewrite: %w`, err))
		}
	}
	return nil
}
