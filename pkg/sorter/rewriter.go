package sorter

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"strings"
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

// Post-process to ensure blank lines between method declarations

// Pre-allocate with some extra capacity for blank lines

// Check if this line contains a method ending (closing brace)
// and the next line contains a method start

// If current line is a closing brace and next line starts a function

// Add a blank line between methods

// Create a new file structure while preserving all original metadata

// Remove method comments that are already Doc comments

// Add non-method declarations first

// Add methods in sorted order - their Doc comments will move with them

// Simple and safe filtering: only remove comments that are EXACTLY method Doc comments
// to prevent duplication, but keep everything else to avoid breaking comment positioning

// Collect Doc comments from method declarations

// Only filter out comments that are EXACTLY the same as Doc comments

// Keep all comments that are NOT method Doc comments

// Skip comments that are exactly the same as method Doc comments

func WriteFile(filename string, content []byte) error {
	return os.WriteFile(filename, content, 0644)
}

func (s *Sorter) Sort() ([]byte, bool, error) {
	callGraph := buildCallGraph(s.file)
	methods := callGraph.GetMethods()

	if len(methods) == 0 {
		content, err := s.formatFile(s.file)
		return content, false, err
	}

	sortedMethods := sortMethods(methods)

	changed := s.hasOrderChanged(methods, sortedMethods)
	if !changed {
		content, err := s.formatFile(s.file)
		return content, false, err
	}

	newFile := s.reorderMethods(sortedMethods)
	content, err := s.formatFile(newFile)
	return content, true, err
}

func (s *Sorter) ensureBlankLinesBetweenMethods(content string) []byte {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines)+10)

	for i, line := range lines {
		result = append(result, line)

		if i < len(lines)-1 {
			currentTrimmed := strings.TrimSpace(line)
			nextTrimmed := strings.TrimSpace(lines[i+1])

			if currentTrimmed == "}" && strings.HasPrefix(nextTrimmed, "func ") {

				result = append(result, "")
			}
		}
	}

	return []byte(strings.Join(result, "\n"))
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

func (s *Sorter) filterGlobalComments() []*ast.CommentGroup {

	docComments := make(map[*ast.CommentGroup]bool)

	for _, decl := range s.file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Recv != nil && funcDecl.Doc != nil {
			docComments[funcDecl.Doc] = true
		}
	}

	var filteredComments []*ast.CommentGroup
	for _, comment := range s.file.Comments {
		if !docComments[comment] {

			filteredComments = append(filteredComments, comment)
		}

	}

	return filteredComments
}

func (s *Sorter) formatFile(file *ast.File) ([]byte, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, s.fset, file); err != nil {
		return nil, err
	}

	content := buf.String()
	return s.ensureBlankLinesBetweenMethods(content), nil
}

func (s *Sorter) reorderMethods(sortedMethods []*MethodInfo) *ast.File {

	newFile := &ast.File{
		Name:     s.file.Name,
		Doc:      s.file.Doc,
		Package:  s.file.Package,
		Comments: s.filterGlobalComments(),
		Imports:  s.file.Imports,
		Scope:    s.file.Scope,
	}

	methodMap := make(map[*ast.FuncDecl]bool)
	for _, method := range sortedMethods {
		methodMap[method.FuncDecl] = true
	}

	newDecls := make([]ast.Decl, 0, len(s.file.Decls))

	for _, decl := range s.file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && methodMap[funcDecl] {
			continue
		}
		newDecls = append(newDecls, decl)
	}

	for _, method := range sortedMethods {
		newDecls = append(newDecls, method.FuncDecl)
	}

	newFile.Decls = newDecls
	return newFile
}
