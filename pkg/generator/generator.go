/*
Provides a code generator that wraps functional task declarations into
tasks compatible with the build system.
*/
package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"text/template"

	"gitlab.com/kyle_anderson/go-utils/pkg/umath"
)

type taskFunc struct {
	Name   string
	Params []taskParam
}

/* Retrieves the expected name for the task struct corresponding to this task function. */
func (tf *taskFunc) StructName() string {
	return tf.Name + "S"
}

/* Gets the name of the task without the leading "Task". */
func (tf *taskFunc) NameWithoutTask() string {
	return funcDeclMatcher.ReplaceAllString(tf.Name, "")
}

/* Returns the name of the constructor for the task's struct .*/
func (tf *taskFunc) ConstructorName() string {
	return fmt.Sprint("New", tf.Name)
}

type taskParam struct {
	/* Names of the parameters. Multiple names may be here if parameters have the same type. */
	Names []string
	/* Type for the parameters. This type includes the package prefix (if any) followed by the type name. */
	Type string
}

func (tp *taskParam) HashCall(handlerSymbol, selfSymbol string) string {
	builder := strings.Builder{}
	switch tp.Type {
	case "string":
		for _, name := range tp.Names {
			builder.WriteString(fmt.Sprintf("\n\t%s.Write([]byte(%s.%s))", handlerSymbol, selfSymbol, name))
		}
	case "int", "int32", "int64", "uint", "uint32", "uint64", "float32", "float64":
		for _, name := range tp.Names {
			builder.WriteString(fmt.Sprintf("\n\tnbt.HashWriteNum(%s, %s.%s)", handlerSymbol, selfSymbol, name))
		}
	default:
		panic(fmt.Sprint("unsupporded type for hash call generation: ", tp.Type))
	}
	return builder.String()
}

/* Gets the parameter names joined by a comma and space, suitable for being valid Go code. */
func (tp taskParam) JoinedNames() string {
	return strings.Join(tp.Names, ", ")
}

func GenerateTaskType(out io.Writer, task *taskFunc) error {
	/* Note: We assume that the nbt package will be available as "nbt" and has not been renamed.
	Although it would be possible to detect if the user has renamed the package, this would complicate the logic,
	and for now we'd prefer to keep things simple. */
	const errPrefix = `generator.GenerateTaskType: `
	t, err := template.New(`generator`).Parse(`type {{.StructName}} struct {{"{"}}
{{- range .Params  }}
	{{ .JoinedNames }} {{ .Type }}
{{ end -}}
{{"}"}}

func {{ .ConstructorName }}(
{{- range $i, $p := .Params }}
	{{ .JoinedNames }} {{ .Type }},
{{- end }}
) nbt.Task {
	return &{{.StructName}}{
{{- range .Params }}
		{{ .JoinedNames }},
{{- end }}
	}
}

func (t *{{.StructName}}) Perform(h nbt.Handler) error {{"{"}}
	return {{.Name}}(
		h,
{{- range .Params -}}
{{- range .Names }}
		t.{{ . }},
{{- end -}}
{{- end }}
	)
{{"}"}}

func (t *{{.StructName}}) Matches(other nbt.Task) bool {{"{"}}
	if t2, ok := other.(*{{.StructName}}); ok {{"{"}}
		return *t == *t2
	{{"}"}}
	return false
{{"}"}}

func (t *{{.StructName}}) Hash() uint64 {
	h := fnv.New64()
	h.Write([]byte({{ .StructName | printf "%q" }}))
{{- range .Params -}}
{{ .HashCall "h" "t" }}
{{- end }}
	return h.Sum64()
}
`)
	if err != nil {
		return fmt.Errorf(errPrefix+`template parsing error: %w`, err)
	}
	if err := t.Execute(out, task); err != nil {
		return fmt.Errorf(errPrefix+`template execution error: %w`, err)
	}
	return nil
}

const generatedFileExt = "gen"

var generatedFileRegex = regexp.MustCompile(`\.` + generatedFileExt + `\.`)

/*
Determines if the given file is a generated Go source file.
Assumes that the file has already been verified to be a Go source file.
*/
func isGeneratedFile(name string) bool {
	return generatedFileRegex.Match([]byte(name))
}

func parseDir(dir string) (pkgs map[string]*ast.Package, first error) {
	return parser.ParseDir(token.NewFileSet(), dir, func(fi fs.FileInfo) bool { return !isGeneratedFile(fi.Name()) }, 0)
}

/*
Generates the task code for all files in the given package.
An aggregate error, implementing [gitlab.com/kyle_anderson/go-utils/pkg/uerrors.Aggregate]
may be returned if multiple errors were encountered.
*/
func processFiles(pkg *ast.Package) (<-chan *taskFunc, <-chan *ErrFileProcessing) {
	numJobs := umath.Min(runtime.NumCPU()+2, len(pkg.Files))
	taskFuncs := make(chan *taskFunc, numJobs)
	errs := make(chan *ErrFileProcessing, numJobs)

	go func() {
		wg := &sync.WaitGroup{}
		defer func() {
			wg.Wait()
			/* these channels can safely be closed here as all writers should now have terminated. */
			close(taskFuncs)
			close(errs)
		}()
		wg.Add(numJobs)
		jobs := make(chan fileProcessingJob, numJobs)
		for i := 0; i < numJobs; i++ {
			go func() {
				processor(jobs, errs, taskFuncs)
				wg.Done()
			}()
		}
		for filename, file := range pkg.Files {
			jobs <- fileProcessingJob{file, filename}
		}
		close(jobs)
	}()
	return taskFuncs, errs
}

func Generate(inputDirectory string) error {
	const mainPackageName = "main"
	/*
	   - Parse each file, and fore each one:
	   - Gather import information
	   - See if user has renamed the nbt package. If so, use their name for it to avoid possible conflicts.
	   - Perform generation, noting which imports are required. Write output to a temporary file.
	   - With the required imports and number of tasks known, generate the package, imports, and constants in the header of the actual file.
	   - Append the temporary file's contents to the result file.
	   - Done
	*/
	pkgs, err := parseDir(inputDirectory)
	if err != nil {
		return &ErrParse{err}
	}
	mainPkg, ok := pkgs[mainPackageName]
	if !ok {
		return ErrNoMainPackage()
	}
	funcInfos, fileErrs := processFiles(mainPkg)
	allErrs := make(chan error, cap(fileErrs)+1)
	wg := &sync.WaitGroup{}
	defer func() {
		wg.Wait()
		close(allErrs)
	}()
	wg.Add(2)
	go func() {
		defer wg.Done()
		for err := range fileErrs {
			allErrs <- fmt.Errorf(`generator.Generate: file processing error: %w`, err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := createHelperFile(inputDirectory, funcInfos); err != nil {
			allErrs <- fmt.Errorf(`generator.Generate: failed to create helper file: %w`, err)
		}
	}()
	return nil
}
