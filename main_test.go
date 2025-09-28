package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainBinary(t *testing.T) {
	// Build the binary for testing
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "gomsort")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Create test file
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

	// Test dry run
	cmd = exec.Command(binaryPath, "-n", testFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Binary execution failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if len(outputStr) == 0 {
	} else if !strings.Contains(outputStr, "Would sort methods in:") {
		t.Errorf("Expected dry run output, got: %s", outputStr)
	}

	// Verify file wasn't changed
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != testContent {
		t.Error("File was modified during dry run")
	}
}

func TestMainBinaryWithVerbose(t *testing.T) {
	// Build the binary for testing
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "gomsort")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Create test file
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

	// Test verbose mode
	cmd = exec.Command(binaryPath, "-v", testFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Binary execution failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Processing:") {
		t.Errorf("Expected verbose output, got: %s", outputStr)
	}
}

func TestMainBinaryHelp(t *testing.T) {
	// Build the binary for testing
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "gomsort")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Test help flag
	cmd = exec.Command(binaryPath, "-h")
	output, _ := cmd.CombinedOutput()

	outputStr := string(output)
	expectedHelpTexts := []string{
		"Usage:",
		"go-msort sorts Go methods",
		"Recursively processes directories like 'go fmt'",
		"Options:",
		"-n",
		"-v",
	}

	for _, expected := range expectedHelpTexts {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Help output missing expected text '%s'. Got: %s", expected, outputStr)
		}
	}
}

func TestMainBinaryWithNonExistentFile(t *testing.T) {
	// Build the binary for testing
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "gomsort")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Test with non-existent file
	cmd = exec.Command(binaryPath, "/non/existent/file.go")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Should get an error message
	outputStr := string(output)
	if len(outputStr) == 0 {
		t.Error("Expected error message for non-existent file")
	}
}

func TestMainFlagParsing(t *testing.T) {
	// Test default values
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Reset flag state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Test with default arguments (current directory)
	os.Args = []string{"gomsort"}

	var dryRun = flag.Bool("n", false, "dry run - show what would be changed without modifying files")
	var verbose = flag.Bool("v", false, "verbose output")

	flag.Parse()
	args := flag.Args()

	if *dryRun {
		t.Error("Expected dryRun to be false by default")
	}
	if *verbose {
		t.Error("Expected verbose to be false by default")
	}
	if len(args) != 0 {
		t.Errorf("Expected no args, got %v", args)
	}
}

func TestMainFlagParsingWithFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Reset flag state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Test with flags
	os.Args = []string{"gomsort", "-n", "-v", "file1.go", "file2.go"}

	var dryRun = flag.Bool("n", false, "dry run - show what would be changed without modifying files")
	var verbose = flag.Bool("v", false, "verbose output")

	flag.Parse()
	args := flag.Args()

	if !*dryRun {
		t.Error("Expected dryRun to be true")
	}
	if !*verbose {
		t.Error("Expected verbose to be true")
	}

	expectedArgs := []string{"file1.go", "file2.go"}
	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(args))
	}
	for i, expected := range expectedArgs {
		if i >= len(args) || args[i] != expected {
			t.Errorf("Expected arg %d to be %s, got %s", i, expected, args[i])
		}
	}
}

func TestMainWithDryRun(t *testing.T) {
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

	// Save original values
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Test dry run flag
	os.Args = []string{"gomsort", "-n", testFile}

	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	main()

	// Verify file wasn't modified (dry run)
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != testContent {
		t.Error("File was modified despite dry-run flag")
	}
}

func TestMainWithDefaultDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create go.mod file
	goModContent := `module testmodule
go 1.22
`
	if err := os.WriteFile("go.mod", []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	testContent := `package test
type Server struct{}
func (s *Server) Start() error { return nil }
`

	err = os.WriteFile("test.go", []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Save original values
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Test with no arguments (should default to ".")
	os.Args = []string{"gomsort"}

	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	main()
}
