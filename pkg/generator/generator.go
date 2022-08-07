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
	"strings"
	"text/template"
)

/*
	The type used for the task type identifier for generated tasks.

This is used when hashing the task.
*/
type GenTaskId = uint32

type taskFunc struct {
	Name   string
	Params []taskParam
}

/* Retrieves the expected name for the task struct corresponding to this task function. */
func (tf *taskFunc) StructName() string {
	return tf.Name + "S"
}

type taskParam struct {
	/* Names of the parameters. Multiple names may be here if parameters have the same type. */
	Names []string
	/* Type for the parameters. This type includes the package prefix (if any) followed by the type name. */
	Type string
}

/* Gets the parameter names joined by a comma and space, suitable for being valid Go code. */
func (tp taskParam) JoinedNames() string {
	return strings.Join(tp.Names, ", ")
}

func GenerateTaskType(out io.Writer, task *taskFunc) error {
	// TODO consider: instead of generating a type for each task function, create a general task type
	// and in the constructor for the task, fill it in.
	/* General task type:
	type generalTask[Args any] struct {
		identifier uint32 // Identifier for the type of the task
		args Args
	}

	Advantages:
	  - Avoids exposing underlying task type to user, who may construct the task raw without the `New<taskname>` constructor
	Disadvantages:
	  - User cannot cast to task type, since none exists. Perhaps though we could accomplish whatever the user wants to achieve from that in other ways,
	    such as with a generated cast function that checks the identifier and returns the values of the arguments provided to the task, which is probably what the user wants.
	*/
	/* TODO: would be a good idea to use the same package name for "nbt" as the code being read
	in case the user renames the package import because of a conflict. */
	const errPrefix = `generator.GenerateTaskType: `
	t, err := template.New(`generator`).Parse(`type {{.StructName}} struct {{"{"}}
	{{ range .Params  }}
	{{ .JoinedNames }} {{ .Type }}
{{ end }}
{{"}"}}

func (t *{{.StructName}}) Perform(h nbt.Handler) {{"{"}}
	{{.Name}}(
		h,
		{{ range .Params }}
		{{ range .Names }}
		t.{{ . }},
		{{ end }}
		{{ end }}
	)
{{"}"}}`)
	if err != nil {
		return fmt.Errorf(errPrefix+`template parsing error: %w`, err)
	}
	if err := t.Execute(out, task); err != nil {
		return fmt.Errorf(errPrefix+`template execution error: %w`, err)
	}
	return nil
}

const generatedFileExt = "gen"

var generatedFileRegex = func() *regexp.Regexp {
	if r, err := regexp.Compile(`\.` + generatedFileExt + `\.`); err != nil {
		panic(fmt.Errorf(`generator.generatedFileRegex: failed to compile: %w`, err))
	} else {
		return r
	}
}()

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

func gatherImportInfo(pkg *ast.Package) {
}

func Generate(inputDirectory string, output io.Writer) error {
	const mainPackageName = "main"
	/*
	   - Parse files
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
	fmt.Printf("mainPkg: %v\n", mainPkg)
	return nil
}
