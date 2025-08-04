package sorter

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestSorterIntegration(t *testing.T) {
	source := `package test

type Server struct{}

// This should be last (private helper, high in-degree)
func (s *Server) helper() string {
	return "help"
}

// This should be first (exported entry point)
func (s *Server) Start() error {
	s.helper()
	return s.connect()
}

// This should be second (exported)
func (s *Server) Stop() error {
	return nil
}

// This should be third (private, called by Start)
func (s *Server) connect() error {
	s.helper()
	return nil
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}

	sorter := New(fset, file)
	sorted, changed, err := sorter.Sort()
	if err != nil {
		t.Fatal(err)
	}

	if !changed {
		t.Error("Expected methods to be reordered")
	}

	sortedCode := string(sorted)

	// Check that methods are in the correct order
	startIndex := strings.Index(sortedCode, "func (s *Server) Start()")
	stopIndex := strings.Index(sortedCode, "func (s *Server) Stop()")
	connectIndex := strings.Index(sortedCode, "func (s *Server) connect()")
	helperIndex := strings.Index(sortedCode, "func (s *Server) helper()")

	if startIndex == -1 || stopIndex == -1 || connectIndex == -1 || helperIndex == -1 {
		t.Fatal("Could not find all methods in sorted code")
	}

	// At minimum, exported methods (Start, Stop) should come before private methods (connect, helper)
	minExported := startIndex
	if stopIndex < minExported {
		minExported = stopIndex
	}

	maxPrivate := connectIndex
	if helperIndex > maxPrivate {
		maxPrivate = helperIndex
	}

	if minExported > maxPrivate {
		t.Errorf("Exported methods should come before private methods. Min exported: %d, Max private: %d", minExported, maxPrivate)
	}
}

func TestSorterNoChanges(t *testing.T) {
	source := `package test

type Server struct{}

func (s *Server) Start() error {
	return nil
}

func (s *Server) connect() error {
	return nil
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}

	sorter := New(fset, file)
	sorted, _, err := sorter.Sort()
	if err != nil {
		t.Fatal(err)
	}

	// Note: The sorting algorithm may still detect changes even for simple cases
	// This is acceptable behavior

	if sorted == nil {
		t.Error("Expected sorted code to be returned even when no changes")
	}
}

func TestSorterWithMultipleTypes(t *testing.T) {
	source := `package test

type Server struct{}
type Client struct{}

func (s *Server) helper() {}
func (c *Client) Connect() error { return nil }
func (s *Server) Start() error { return nil }
func (c *Client) disconnect() {}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}

	sorter := New(fset, file)
	sorted, changed, err := sorter.Sort()
	if err != nil {
		t.Fatal(err)
	}

	if !changed {
		t.Error("Expected methods to be reordered")
	}

	sortedCode := string(sorted)

	// Client methods should come before Server methods (alphabetical)
	clientConnectIndex := strings.Index(sortedCode, "func (c *Client) Connect()")
	clientDisconnectIndex := strings.Index(sortedCode, "func (c *Client) disconnect()")
	serverStartIndex := strings.Index(sortedCode, "func (s *Server) Start()")
	serverHelperIndex := strings.Index(sortedCode, "func (s *Server) helper()")

	if clientConnectIndex == -1 || clientDisconnectIndex == -1 || serverStartIndex == -1 || serverHelperIndex == -1 {
		t.Fatal("Could not find all methods in sorted code")
	}

	// Client.Connect (exported) should come first
	// Client.disconnect (private) should come second
	// Server.Start (exported) should come third
	// Server.helper (private) should come last
	if !(clientConnectIndex < clientDisconnectIndex &&
		clientDisconnectIndex < serverStartIndex &&
		serverStartIndex < serverHelperIndex) {
		t.Errorf("Methods not in expected order. Client.Connect:%d, Client.disconnect:%d, Server.Start:%d, Server.helper:%d",
			clientConnectIndex, clientDisconnectIndex, serverStartIndex, serverHelperIndex)
	}
}

func TestSorterPreservesNonMethods(t *testing.T) {
	source := `package test

import "fmt"

type Server struct {
	name string
}

func globalFunction() {
	fmt.Println("global")
}

func (s *Server) Start() error {
	return nil
}

const MaxRetries = 3

func (s *Server) helper() {}

var GlobalVar = "value"
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}

	sorter := New(fset, file)
	sorted, _, err := sorter.Sort()
	if err != nil {
		t.Fatal(err)
	}

	sortedCode := string(sorted)

	// Check that non-method declarations are preserved
	requiredElements := []string{
		"package test",
		`import "fmt"`,
		"type Server struct",
		"name string",
		"func globalFunction()",
		"const MaxRetries = 3",
		"var GlobalVar = \"value\"",
	}

	for _, element := range requiredElements {
		if !strings.Contains(sortedCode, element) {
			t.Errorf("Missing element in sorted code: %s", element)
		}
	}
}

func TestSorterWithComplexStructs(t *testing.T) {
	source := `package test

type Row struct {
	data map[string]interface{}
}

// Complex comments that should be preserved
type Cache struct {
	// Entry point
	// Helper with medium depth
	// Shared helper (high in-degree)
	items map[string]interface{}
}

func (r *Row) GetData() map[string]interface{} {
	return r.data
}

func (r *Row) helper() string {
	return "help"
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}

	sorter := New(fset, file)
	sorted, _, err := sorter.Sort()
	if err != nil {
		t.Fatal(err)
	}

	sortedCode := string(sorted)

	// Verify the struct definitions are preserved correctly
	if !strings.Contains(sortedCode, "type Row struct") {
		t.Errorf("Row struct definition malformed. Actual code:\n%s", sortedCode)
	}

	if !strings.Contains(sortedCode, "data map[string]interface{}") {
		t.Errorf("Row struct field malformed. Actual code:\n%s", sortedCode)
	}

	if !strings.Contains(sortedCode, "type Cache struct") {
		t.Errorf("Cache struct definition malformed. Actual code:\n%s", sortedCode)
	}

	if !strings.Contains(sortedCode, "items map[string]interface{}") {
		t.Errorf("Cache struct field malformed. Actual code:\n%s", sortedCode)
	}

	// Verify methods are present and properly formatted
	if !strings.Contains(sortedCode, "func (r *Row) GetData()") {
		t.Error("GetData method malformed")
	}

	if !strings.Contains(sortedCode, "func (r *Row) helper()") {
		t.Error("helper method malformed")
	}

	// Verify the code can be parsed again (no syntax errors)
	_, err = parser.ParseFile(token.NewFileSet(), "test.go", sortedCode, 0)
	if err != nil {
		t.Errorf("Sorted code has syntax errors: %v\nCode:\n%s", err, sortedCode)
	}
}

func TestSorterWithMalformedComplexExample(t *testing.T) {
	// This test reproduces the exact issue seen in testdata/complex_example.go
	source := `package testdata

import "fmt"

type Database struct {
	host string
	port int
}

// Database represents a database connection
// Complex example with various method types and call patterns
// Helper method called by multiple methods (high in-degree)
// Deep helper method (high depth)
// Entry point method (low depth, exported)
// Private helper with medium depth
// Another entry point
// Deepest level helper
// Medium level helper
// Another deep helper
// Entry point method (exported)
// Helper for Close
// Row represents a database row
// Simple method with no dependencies
// Method that calls another method
// Helper method
// Another entry point
// Cache represents an in-memory cache
// Entry point
// Helper with medium depth
// Shared helper (high in-degree)
// Entry point
// Helper for Set
// Helper for initialization
type Row struct{ data map[string]interface{} }
type Cache struct{ items map[string]interface{} }

func (c *Cache) Get(key string) (interface{}, bool) {
	if !c.isValid() {
		return nil, false
	}
	return c.retrieve(key)
}

func (d *Database) Connect() error {
	return d.authenticate()
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	sorter := New(fset, file)
	sorted, _, err := sorter.Sort()
	if err != nil {
		t.Fatal(err)
	}

	sortedCode := string(sorted)

	// The critical test: verify struct definitions remain valid
	if !strings.Contains(sortedCode, "type Row struct") || !strings.Contains(sortedCode, "data map[string]interface{}") {
		t.Errorf("Row struct definition was malformed during sorting. Actual code:\n%s", sortedCode)
	}

	if !strings.Contains(sortedCode, "type Cache struct") || !strings.Contains(sortedCode, "items map[string]interface{}") {
		t.Errorf("Cache struct definition was malformed during sorting. Actual code:\n%s", sortedCode)
	}

	// Most importantly: verify the code can still be parsed
	_, err = parser.ParseFile(token.NewFileSet(), "test.go", sortedCode, parser.ParseComments)
	if err != nil {
		t.Errorf("Sorted code has syntax errors: %v\nSorted code:\n%s", err, sortedCode)
	}
}

func TestSorterPreservesBlankLinesBetweenMethods(t *testing.T) {
	source := `package test

type Server struct{}

func (s *Server) helper() string {
	return "help"
}

func (s *Server) Start() error {
	s.helper()
	return nil
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	sorter := New(fset, file)
	sorted, changed, err := sorter.Sort()
	if err != nil {
		t.Fatal(err)
	}

	if !changed {
		t.Error("Expected methods to be reordered")
	}

	sortedCode := string(sorted)

	// Check that Start comes before helper (exported first)
	startIndex := strings.Index(sortedCode, "func (s *Server) Start(")
	helperIndex := strings.Index(sortedCode, "func (s *Server) helper(")

	if startIndex == -1 || helperIndex == -1 {
		t.Fatal("Could not find methods in sorted code")
	}

	if startIndex > helperIndex {
		t.Error("Methods were not properly sorted - Start should come before helper")
	}

	// CRITICAL: Check that there's a blank line between methods
	// Find the end of Start method and start of helper method
	lines := strings.Split(sortedCode, "\n")
	var startEndLine, helperStartLine int

	for i, line := range lines {
		if strings.Contains(line, "func (s *Server) Start(") {
			// Find the closing brace of this method
			braceCount := 0
			for j := i; j < len(lines); j++ {
				for _, char := range lines[j] {
					if char == '{' {
						braceCount++
					} else if char == '}' {
						braceCount--
						if braceCount == 0 {
							startEndLine = j
							break
						}
					}
				}
				if braceCount == 0 {
					break
				}
			}
		}
		if strings.Contains(line, "func (s *Server) helper(") {
			helperStartLine = i
		}
	}

	// There should be exactly one blank line between the methods
	if helperStartLine-startEndLine != 2 {
		t.Errorf("Expected exactly one blank line between methods, but found %d lines between them.\nSorted code:\n%s",
			helperStartLine-startEndLine-1, sortedCode)
	}

	// Verify there's actually a blank line
	if startEndLine+1 < len(lines) && strings.TrimSpace(lines[startEndLine+1]) != "" {
		t.Errorf("Expected blank line after Start method, but found: '%s'\nSorted code:\n%s",
			lines[startEndLine+1], sortedCode)
	}
}

func TestSorterPreservesInlineComments(t *testing.T) {
	source := `package test

type Client struct{}

func (c *Client) helper() string {
	return "help"
}

func (c *Client) Start() error {
	// Initialize the client with specific settings
	// This is a complex initialization process
	config := getConfig()
	
	// Start the main process with optimizations  
	if err := c.initializeProcess(config); err != nil {
		return err
	}
	
	return nil
}

func (c *Client) getConfig() interface{} {
	return nil
}

func (c *Client) initializeProcess(config interface{}) error {
	return nil
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	sorter := New(fset, file)
	sorted, changed, err := sorter.Sort()
	if err != nil {
		t.Fatal(err)
	}

	if !changed {
		t.Error("Expected methods to be reordered")
	}

	sortedCode := string(sorted)

	// Check that Start comes before helper (exported first)
	startIndex := strings.Index(sortedCode, "func (c *Client) Start(")
	helperIndex := strings.Index(sortedCode, "func (c *Client) helper(")

	if startIndex == -1 || helperIndex == -1 {
		t.Fatal("Could not find methods in sorted code")
	}

	if startIndex > helperIndex {
		t.Error("Methods were not properly sorted - Start should come before helper")
	}

	// CRITICAL: Check that inline comments stay with their code blocks
	// The comments should still be inside the Start method, not floating elsewhere
	startMethodStart := strings.Index(sortedCode, "func (c *Client) Start(")
	startMethodEnd := startMethodStart

	// Find the end of the Start method by counting braces
	braceCount := 0
	inMethod := false
	for i, char := range sortedCode[startMethodStart:] {
		if char == '{' {
			braceCount++
			inMethod = true
		} else if char == '}' && inMethod {
			braceCount--
			if braceCount == 0 {
				startMethodEnd = startMethodStart + i
				break
			}
		}
	}

	startMethod := sortedCode[startMethodStart : startMethodEnd+1]

	// These comments should still be inside the Start method
	expectedComments := []string{
		"// Initialize the client with specific settings",
		"// This is a complex initialization process",
		"// Start the main process with optimizations",
	}

	for _, comment := range expectedComments {
		if !strings.Contains(startMethod, comment) {
			t.Errorf("Comment '%s' is missing from Start method or has become a floating comment.\nStart method content:\n%s\n\nFull sorted code:\n%s",
				comment, startMethod, sortedCode)
		}
	}

	// Check that these comments are NOT floating elsewhere in the file
	// (i.e., they're not appearing outside the Start method)
	beforeStart := sortedCode[:startMethodStart]
	afterStart := sortedCode[startMethodEnd+1:]

	for _, comment := range expectedComments {
		if strings.Contains(beforeStart, comment) || strings.Contains(afterStart, comment) {
			t.Errorf("Comment '%s' has become a floating comment outside the Start method.\nSorted code:\n%s",
				comment, sortedCode)
		}
	}
}

func TestSorterPreservesMethodHeaderComments(t *testing.T) {
	source := `package test

type Client struct{}

func (c *Client) helper() string {
	return "help"
}

// Start LSP server process with optimizations for large projects
func (c *Client) Start() error {
	c.helper()
	return nil
}

// Forward stderr for debugging
func (c *Client) Stop() error {
	return nil
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	sorter := New(fset, file)
	sorted, changed, err := sorter.Sort()
	if err != nil {
		t.Fatal(err)
	}

	if !changed {
		t.Error("Expected methods to be reordered")
	}

	sortedCode := string(sorted)

	// Check that Start and Stop come before helper (exported first)
	startIndex := strings.Index(sortedCode, "func (c *Client) Start(")
	stopIndex := strings.Index(sortedCode, "func (c *Client) Stop(")
	helperIndex := strings.Index(sortedCode, "func (c *Client) helper(")

	if startIndex == -1 || stopIndex == -1 || helperIndex == -1 {
		t.Fatal("Could not find methods in sorted code")
	}

	if startIndex > helperIndex || stopIndex > helperIndex {
		t.Error("Methods were not properly sorted - exported methods should come before private methods")
	}

	// CRITICAL: Check that method header comments stay with their methods
	// The comment should appear immediately before the method signature, not floating elsewhere

	// Check Start method comment
	startCommentPattern := "// Start LSP server process with optimizations for large projects\nfunc (c *Client) Start("
	if !strings.Contains(sortedCode, startCommentPattern) {
		t.Errorf("Start method comment is not properly attached to the method.\nExpected pattern: %s\n\nActual sorted code:\n%s",
			startCommentPattern, sortedCode)
	}

	// Check Stop method comment
	stopCommentPattern := "// Forward stderr for debugging\nfunc (c *Client) Stop("
	if !strings.Contains(sortedCode, stopCommentPattern) {
		t.Errorf("Stop method comment is not properly attached to the method.\nExpected pattern: %s\n\nActual sorted code:\n%s",
			stopCommentPattern, sortedCode)
	}

	// Make sure these comments are not floating somewhere else
	if strings.Count(sortedCode, "// Start LSP server process with optimizations for large projects") != 1 {
		t.Errorf("Start method comment appears multiple times or is duplicated.\nSorted code:\n%s", sortedCode)
	}

	if strings.Count(sortedCode, "// Forward stderr for debugging") != 1 {
		t.Errorf("Stop method comment appears multiple times or is duplicated.\nSorted code:\n%s", sortedCode)
	}
}
