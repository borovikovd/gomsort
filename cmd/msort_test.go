package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWithDryRun(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	testContent := `package test

type Server struct{}

func (s *Server) helper() {}
func (s *Server) Start() error { return nil }
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		DryRun:  true,
		Verbose: false,
		Paths:   []string{testFile},
	}

	err = Run(config)
	if err != nil {
		t.Errorf("Run() failed: %v", err)
	}

	// Verify file wasn't modified
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != testContent {
		t.Error("File was modified despite dry-run flag")
	}
}

func TestRunWithActualSort(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	testContent := `package test

type Server struct{}

func (s *Server) helper() {}
func (s *Server) Start() error { return nil }
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		DryRun:  false,
		Verbose: false,
		Paths:   []string{testFile},
	}

	err = Run(config)
	if err != nil {
		t.Errorf("Run() failed: %v", err)
	}

	// Verify file was modified
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	modifiedContent := string(content)

	// Start() should come before helper()
	startIndex := strings.Index(modifiedContent, "func (s *Server) Start()")
	helperIndex := strings.Index(modifiedContent, "func (s *Server) helper()")

	if startIndex == -1 || helperIndex == -1 {
		t.Fatal("Could not find methods in modified file")
	}

	if startIndex > helperIndex {
		t.Error("Methods were not properly sorted")
	}
}

func TestRunWithDirectory(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create go.mod file
	goModContent := `module testmodule

go 1.22
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create multiple Go files
	testFiles := []struct {
		name    string
		content string
	}{
		{
			"server.go",
			`package test
type Server struct{}
func (s *Server) helper() {}
func (s *Server) Start() error { return nil }
`,
		},
		{
			"client.go",
			`package test
type Client struct{}
func (c *Client) helper() {}
func (c *Client) Connect() error { return nil }
`,
		},
		{
			"not_go.txt",
			"This should be ignored",
		},
		{
			"test_file_test.go",
			`package test
func TestSomething(t *testing.T) {}
`,
		},
	}

	for _, tf := range testFiles {
		path := filepath.Join(tmpDir, tf.name)
		err := os.WriteFile(path, []byte(tf.content), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	config := &Config{
		DryRun:  false,
		Verbose: true,
		Paths:   []string{tmpDir},
	}

	err := Run(config)
	if err != nil {
		t.Errorf("Run() failed: %v", err)
	}

	// Verify Go files were processed (excluding test files)
	for _, tf := range testFiles {
		if strings.HasSuffix(tf.name, ".go") && !strings.HasSuffix(tf.name, "_test.go") {
			path := filepath.Join(tmpDir, tf.name)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			modifiedContent := string(content)

			// Basic validation - just check that file was processed
			if len(modifiedContent) == 0 {
				t.Errorf("File %s appears to be empty after processing", tf.name)
			}
		}
	}
}

func TestRunWithRecursive(t *testing.T) {
	// Create a nested directory structure
	tmpDir := t.TempDir()

	// Create go.mod file
	goModContent := `module testmodule

go 1.22
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create files in both directories
	testContent := `package test
type Server struct{}
func (s *Server) helper() {}
func (s *Server) Start() error { return nil }
`

	rootFile := filepath.Join(tmpDir, "root.go")
	subFile := filepath.Join(subDir, "sub.go")

	err = os.WriteFile(rootFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(subFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		DryRun:  false,
		Verbose: false,
		Paths:   []string{tmpDir},
	}

	err = Run(config)
	if err != nil {
		t.Errorf("Run() failed: %v", err)
	}

	// Verify both files were processed
	for _, file := range []string{rootFile, subFile} {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		modifiedContent := string(content)
		// Basic validation that file was processed
		if len(modifiedContent) == 0 {
			t.Errorf("File %s appears to be empty after processing", file)
		}
	}
}

func TestRunWithNonExistentFile(t *testing.T) {
	config := &Config{
		DryRun:  false,
		Verbose: false,
		Paths:   []string{"/non/existent/file.go"},
	}

	err := Run(config)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestRunWithInvalidGoFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.go")

	// Create invalid Go syntax
	invalidContent := `package test
func (s *Server invalid syntax here
`

	err := os.WriteFile(testFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		DryRun:  false,
		Verbose: false,
		Paths:   []string{testFile},
	}

	err = Run(config)
	if err == nil {
		t.Error("Expected error for invalid Go file")
	}
}
