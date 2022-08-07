package generator

import (
	"fmt"
	"go/ast"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"gitlab.com/kyle_anderson/go-utils/pkg/set"
)

type fileOutputter struct {
	body  *os.File
	final *os.File
}

var extensionReplacer = regexp.MustCompile(`\.go$`)

/*
Creates a new file outputter for the given source file name, and will use the given
working directory for any temporary files.
workingDir is assumed to be a valid path to an existing directory in which temporary files
can be placed.

TODO: can probably just write the body content to memory instead of a temporary file.
*/
func newFileOutputter(sourceFileName, workingDir string) (*fileOutputter, error) {
	outFileName := extensionReplacer.ReplaceAllString(sourceFileName, "."+generatedFileExt+".go")
	body, err := os.CreateTemp(workingDir, filepath.Base(outFileName))
	if err != nil {
		return nil, fmt.Errorf(`generator.newFileOutputter: failed to open temporary file: %w`, err)
	}
	final, err := os.Create(outFileName)
	if err != nil {
		return nil, fmt.Errorf(`generator.newFileOutputter: failed to create final file: %w`, err)
	}
	return &fileOutputter{body, final}, nil
}

func (o *fileOutputter) Body() io.Writer   { return o.body }
func (o *fileOutputter) Header() io.Writer { return o.final }
func (o *fileOutputter) Close() error {
	defer func() {
		o.body.Close()
		o.final.Close()
	}()
	if _, err := o.body.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf(`generator.(*fileOutputter).Close: failed to seek to beginning of body: %w`, err)
	}
	_, err := io.Copy(o.final, o.body)
	if err != nil {
		return fmt.Errorf(`generator.(*fileOutputter).Close: failed to copy body to final destination: %w`, err)
	}
	return nil
}

type outputter interface {
	Body() io.Writer
	Header() io.Writer
	Close() error
}

type fileProcessingJob struct {
	file              *ast.File
	outputterSupplier func() (outputter, error)
}

/* Processes incoming file jobs on the given channel. */
func processor(jobs <-chan *fileProcessingJob) {
	for job := range jobs {
		processFile(job)
	}
}

var funcDeclMatcher = regexp.MustCompile(`^Task`)

func nameMatchesTask(decl *ast.FuncDecl) bool {
	return funcDeclMatcher.MatchString(decl.Name.Name)
}
func extractFunctionInformation(fn *ast.FuncDecl) (info *taskFunc, usedPackages set.Set[string]) {
	usedPackages = set.NewComparable[string]()
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
				usedPackage := v.X.(*ast.Ident).Name
				outParam.Type = usedPackage + v.Sel.Name
				usedPackages.Add(usedPackage)
			default:
				panic(fmt.Errorf("unhandled parameter type: %T", v))
			}
		}
	}
	return
}

func getImportName(spec *ast.ImportSpec) (name string, explicitlyNamed bool) {
	if name := spec.Name; name != nil {
		return name.Name, false
	}
	// TODO
	panic("do not yet know a good way to get the name of the import from the path")
	return "", true
}

type importPathInfo struct {
	/* Imported package path. */
	path string
	/* True if the import was explicitly given a name, i.e., if it is of the form `import <name> "path"`. */
	renamed bool
}

func extractImportInformation(f *ast.File) map[string]importPathInfo {
	importInfo := make(map[string]importPathInfo)
	for _, imp := range f.Imports {
		importName, explicitlyNamed := getImportName(imp)
		importInfo[importName] = importPathInfo{imp.Path.Value, explicitlyNamed}
	}
	return importInfo
}

func writeHeader(out io.Writer, usedImports map[string]importPathInfo) error {
	t := template.Must(template.New(`header`).Parse(`package main
import (
	{{range $k, $v := .}}
	{{ if $v.renamed }} {{ $k }} {{ end }} {{$v.path | printf "%q" }
	{{end}}
)`))
	if err := t.Execute(out, usedImports); err != nil {
		return fmt.Errorf(`generator.writeHeader: failed to execute template: %w`, err)
	}
	return nil
}

func processFile(job *fileProcessingJob) error {
	outputter, err := job.outputterSupplier()
	if err != nil {
		return fmt.Errorf(`generator.processFile: error instantiating outputter: %w`, err)
	}
	defer outputter.Close()
	importInfo := extractImportInformation(job.file)
	usedPackages := make(map[string]importPathInfo)
	for _, decl := range job.file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && nameMatchesTask(funcDecl) {
			funcInfo, pkgs := extractFunctionInformation(funcDecl)
			it := pkgs.It()
		loop:
			for {
				elem, err := it.Next()
				switch {
				case err.IsDone():
					break loop
				case err.Unwrap() != nil:
					return fmt.Errorf(`generator.processFile: iteration failed: %w`, err)
				default:
					usedPackages[elem] = importInfo[elem]
				}
			}
			if err := GenerateTaskType(outputter.Body(), funcInfo); err != nil {
				return fmt.Errorf(`generator.processFile: failed to generate task type: %w`, err)
			}
		}
	}
	if err := writeHeader(outputter.Header(), usedPackages); err != nil {
		return fmt.Errorf(`generator.processFile: failed to write header: %w`, err)
	}
	return nil
}
