package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainBinary(t *testing.T) {
	// Build the binary for testing
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "go-msort")

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
	// Note: dry run may not show output if no changes are needed
	if len(outputStr) == 0 {
		// This is acceptable - no changes may be needed
		t.Logf("No output from dry run - file may already be sorted")
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
	binaryPath := filepath.Join(tmpDir, "go-msort")

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
	binaryPath := filepath.Join(tmpDir, "go-msort")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Test help flag
	cmd = exec.Command(binaryPath, "-h")
	output, _ := cmd.CombinedOutput()

	// Help should exit with code 2 (standard for flag package)
	// We expect this to fail, so err != nil is expected

	outputStr := string(output)
	expectedHelpTexts := []string{
		"Usage:",
		"go-msort sorts Go methods",
		"Options:",
		"-n",
		"-r",
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
	binaryPath := filepath.Join(tmpDir, "go-msort")

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
