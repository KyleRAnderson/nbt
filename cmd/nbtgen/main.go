package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"regexp"
)

func main() {
	fset := token.NewFileSet()
	pkgs, firstErr := parser.ParseDir(fset, "pkg/generator/test_cases/1", func(fi fs.FileInfo) bool {
		if matched, err := regexp.Match(`\.gen\.`, []byte(fi.Name())); err == nil {
			return !matched
		} else {
			panic(fmt.Errorf("error matching filenames: %w", err))
		}
	}, 0)
	if firstErr != nil {
		log.Fatal(firstErr)
	}
	/* We only care about the main package (for now anyway, may expand on this later). */
	mainPkg := pkgs["main"]
	fmt.Printf("mainPkg: %#v\n", mainPkg)
	for _, ideaFile := range mainPkg.Files {
		fmt.Printf("ideaFile: %#v\n", ideaFile)
		fmt.Println("declarations: ")
		for _, decl := range ideaFile.Decls {
			fmt.Printf("decl: %#v\n", decl)
			t, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			for _, param := range t.Type.Params.List {
				fmt.Printf("param.Type: %v, Type: %T\n", param.Type, param.Type)
				if id, ok := param.Type.(*ast.Ident); ok {
					fmt.Printf("id.Name: %v\n", id.Name)
				} else if sel, ok := param.Type.(*ast.SelectorExpr); ok {
					fmt.Printf("sel.X: %#v\n", sel.X)
					fmt.Printf("sel.Sel: %v\n", sel.Sel)
				}
				fmt.Printf("param.Names: %v\n", param.Names)
			}
		}
	}
}
