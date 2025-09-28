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

	// Create go.mod file
	goModContent := `module testmodule

go 1.22
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

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

	// Create go.mod file
	goModContent := `module testmodule

go 1.22
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

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

func TestProcessPathWithNonGoFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(testFile, []byte("not go content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		DryRun:  false,
		Verbose: false,
		Paths:   []string{testFile},
	}

	// Should not error but also should not process
	err = Run(config)
	if err != nil {
		t.Errorf("Run() with non-Go file should not error: %v", err)
	}
}

func TestProcessPathWithTestFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_test.go")

	testContent := `package test
func TestSomething(t *testing.T) {}
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

	// Should not error but also should not process test files
	err = Run(config)
	if err != nil {
		t.Errorf("Run() with test file should not error: %v", err)
	}

	// Verify file wasn't modified
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != testContent {
		t.Error("Test file should not be modified")
	}
}

func TestCheckGoModuleWithoutGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		DryRun:  false,
		Verbose: false,
		Paths:   []string{tmpDir},
	}

	err := Run(config)
	if err == nil {
		t.Error("Expected error when no go.mod found")
	}

	expectedErrMsg := "go.mod file not found"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error about go.mod, got: %v", err)
	}
}

func TestProcessDirectoryWithHiddenDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod file
	goModContent := `module testmodule
go 1.22
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create hidden directory
	hiddenDir := filepath.Join(tmpDir, ".hidden")
	err := os.Mkdir(hiddenDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create a Go file in the hidden directory
	hiddenFile := filepath.Join(hiddenDir, "hidden.go")
	hiddenContent := `package hidden
type Server struct{}
func (s *Server) helper() {}
func (s *Server) Start() error { return nil }
`
	err = os.WriteFile(hiddenFile, []byte(hiddenContent), 0644)
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
		t.Errorf("Run() should not error with hidden directories: %v", err)
	}

	// Verify hidden file was not modified
	content, err := os.ReadFile(hiddenFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != hiddenContent {
		t.Error("Files in hidden directories should not be processed")
	}
}

func TestProcessFileWithVerboseNoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "sorted.go")

	// Create go.mod file
	goModContent := `module testmodule
go 1.22
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create already sorted content
	sortedContent := `package test

type Server struct{}

// Start is the entry point
func (s *Server) Start() error {
	return s.helper()
}

// helper is a private helper
func (s *Server) helper() error {
	return nil
}
`

	err := os.WriteFile(testFile, []byte(sortedContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		DryRun:  false,
		Verbose: true,
		Paths:   []string{testFile},
	}

	err = Run(config)
	if err != nil {
		t.Errorf("Run() failed: %v", err)
	}

	// Verify file wasn't modified since it was already sorted
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != sortedContent {
		t.Error("Already sorted file should not be modified")
	}
}

func TestProcessDirectoryWithNestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	goModContent := `module testmodule
go 1.22
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	level1Dir := filepath.Join(tmpDir, "level1")
	level2Dir := filepath.Join(level1Dir, "level2")

	if err := os.MkdirAll(level2Dir, 0755); err != nil {
		t.Fatal(err)
	}

	testContent := `package test

type Server struct{}

func (s *Server) helper() error {
	return nil
}

func (s *Server) Start() error {
	return s.helper()
}
`

	rootFile := filepath.Join(tmpDir, "root.go")
	if err := os.WriteFile(rootFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	level1File := filepath.Join(level1Dir, "level1.go")
	if err := os.WriteFile(level1File, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	level2File := filepath.Join(level2Dir, "level2.go")
	if err := os.WriteFile(level2File, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
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

	filesToCheck := []string{rootFile, level1File, level2File}
	for _, file := range filesToCheck {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		modifiedContent := string(content)
		startIndex := strings.Index(modifiedContent, "func (s *Server) Start()")
		helperIndex := strings.Index(modifiedContent, "func (s *Server) helper()")

		if startIndex == -1 || helperIndex == -1 {
			t.Fatalf("Could not find methods in file %s", file)
		}

		if startIndex > helperIndex {
			t.Errorf("Methods were not properly sorted in file %s", file)
		}
	}
}

func TestProcessDirectoryReadError(t *testing.T) {
	config := &Config{
		DryRun:  false,
		Verbose: false,
		Paths:   []string{"/non/existent/directory/that/should/not/exist"},
	}

	err := Run(config)
	if err == nil {
		t.Error("Expected error when processing non-existent directory")
	}
}
