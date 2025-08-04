package sorter

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"os"
)

type Sorter struct {
	fset *token.FileSet
	file *ast.File
}

func New(fset *token.FileSet, file *ast.File) *Sorter {
	return &Sorter{
		fset: fset,
		file: file,
	}
}

func (s *Sorter) Sort() ([]byte, bool, error) {
	callGraph := buildCallGraph(s.file)
	methods := callGraph.GetMethods()

	if len(methods) == 0 {
		var buf bytes.Buffer
		if err := format.Node(&buf, s.fset, s.file); err != nil {
			return nil, false, err
		}
		return buf.Bytes(), false, nil
	}

	sortedMethods := sortMethods(methods)

	changed := s.hasOrderChanged(methods, sortedMethods)
	if !changed {
		var buf bytes.Buffer
		if err := format.Node(&buf, s.fset, s.file); err != nil {
			return nil, false, err
		}
		return buf.Bytes(), false, nil
	}

	newFile := s.reorderMethods(sortedMethods)

	var buf bytes.Buffer
	if err := format.Node(&buf, s.fset, newFile); err != nil {
		return nil, false, err
	}

	return buf.Bytes(), true, nil
}

func (s *Sorter) hasOrderChanged(original, sorted []*MethodInfo) bool {
	if len(original) != len(sorted) {
		return true
	}

	for i, method := range original {
		if method.Position != sorted[i].Position {
			return true
		}
	}

	return false
}

func (s *Sorter) reorderMethods(sortedMethods []*MethodInfo) *ast.File {
	newFile := &ast.File{
		Name:     s.file.Name,
		Doc:      s.file.Doc,
		Package:  s.file.Package,
		Comments: s.file.Comments,
		Imports:  s.file.Imports,
		Scope:    s.file.Scope,
	}

	methodMap := make(map[*ast.FuncDecl]bool)
	for _, method := range sortedMethods {
		methodMap[method.FuncDecl] = true
	}

	for _, decl := range s.file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if methodMap[funcDecl] {
				continue
			}
		}
		newFile.Decls = append(newFile.Decls, decl)
	}

	for _, method := range sortedMethods {
		newFile.Decls = append(newFile.Decls, method.FuncDecl)
	}

	return newFile
}

func WriteFile(filename string, content []byte) error {
	return os.WriteFile(filename, content, 0644)
}
