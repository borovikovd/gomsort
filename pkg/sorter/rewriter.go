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

func (s *Sorter) formatFile(file *ast.File) ([]byte, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, s.fset, file); err != nil {
		return nil, err
	}

	// Post-process to ensure blank lines between method declarations
	content := buf.String()
	return s.ensureBlankLinesBetweenMethods(content), nil
}

// ensureBlankLinesBetweenMethods adds blank lines between consecutive method declarations
func (s *Sorter) ensureBlankLinesBetweenMethods(content string) []byte {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines)+10) // Pre-allocate with some extra capacity for blank lines

	for i, line := range lines {
		result = append(result, line)

		// Check if this line contains a method ending (closing brace)
		// and the next line contains a method start
		if i < len(lines)-1 {
			currentTrimmed := strings.TrimSpace(line)
			nextTrimmed := strings.TrimSpace(lines[i+1])

			// If current line is a closing brace and next line starts a function
			if currentTrimmed == "}" && strings.HasPrefix(nextTrimmed, "func ") {
				// Add a blank line between methods
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

func (s *Sorter) reorderMethods(sortedMethods []*MethodInfo) *ast.File {
	// Create a new file structure while preserving all original metadata
	newFile := &ast.File{
		Name:     s.file.Name,
		Doc:      s.file.Doc,
		Package:  s.file.Package,
		Comments: s.filterGlobalComments(), // Remove method comments that are already Doc comments
		Imports:  s.file.Imports,
		Scope:    s.file.Scope,
	}

	methodMap := make(map[*ast.FuncDecl]bool)
	for _, method := range sortedMethods {
		methodMap[method.FuncDecl] = true
	}

	newDecls := make([]ast.Decl, 0, len(s.file.Decls))

	// Add non-method declarations first
	for _, decl := range s.file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && methodMap[funcDecl] {
			continue
		}
		newDecls = append(newDecls, decl)
	}

	// Add methods in sorted order - their Doc comments will move with them
	for _, method := range sortedMethods {
		newDecls = append(newDecls, method.FuncDecl)
	}

	newFile.Decls = newDecls
	return newFile
}

// filterGlobalComments removes comments that are already associated with method Doc fields
// to prevent duplicate comments in the output
func (s *Sorter) filterGlobalComments() []*ast.CommentGroup {
	// Smart filtering: only remove method header comments that are likely to be repositioned
	// incorrectly, while keeping others that go/format needs for proper positioning

	docComments := make(map[*ast.CommentGroup]bool)

	// Collect Doc comments from method declarations
	for _, decl := range s.file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Recv != nil && funcDecl.Doc != nil {
			docComments[funcDecl.Doc] = true
		}
	}

	var filteredComments []*ast.CommentGroup
	for _, comment := range s.file.Comments {
		// Only filter out Doc comments that are directly before method declarations
		// Keep inline comments and other comments that go/format needs
		if docComments[comment] {
			// This is a method header comment - check if it would cause floating
			shouldFilter := s.isMethodHeaderComment(comment)
			if !shouldFilter {
				filteredComments = append(filteredComments, comment)
			}
		} else {
			// Keep all non-Doc comments (inline comments, etc.)
			filteredComments = append(filteredComments, comment)
		}
	}

	return filteredComments
}

// isMethodHeaderComment determines if a comment is a method header that should be filtered
func (s *Sorter) isMethodHeaderComment(comment *ast.CommentGroup) bool {
	commentEnd := s.fset.Position(comment.End()).Line

	// Find if there's a method declaration immediately after this comment
	for _, decl := range s.file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Recv != nil {
			declStart := s.fset.Position(decl.Pos()).Line

			// If method starts within 2 lines after comment ends, it's likely a header comment
			if declStart > commentEnd && declStart-commentEnd <= 2 {
				return true
			}
		}
	}

	return false
}

func WriteFile(filename string, content []byte) error {
	return os.WriteFile(filename, content, 0644)
}
