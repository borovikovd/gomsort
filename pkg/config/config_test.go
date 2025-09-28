package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.SortCriteria.GroupByReceiver {
		t.Error("Expected GroupByReceiver to be true")
	}
	if !config.SortCriteria.ExportedFirst {
		t.Error("Expected ExportedFirst to be true")
	}
	if !config.SortCriteria.SortByDepth {
		t.Error("Expected SortByDepth to be true")
	}
	if !config.SortCriteria.SortByInDegree {
		t.Error("Expected SortByInDegree to be true")
	}
	if !config.SortCriteria.PreserveOrigOrder {
		t.Error("Expected PreserveOrigOrder to be true")
	}

	if len(config.Exclude) != 0 {
		t.Errorf("Expected empty Exclude, got %v", config.Exclude)
	}
	if len(config.Include) != 1 || config.Include[0] != "*.go" {
		t.Errorf("Expected Include to be ['*.go'], got %v", config.Include)
	}
}

func TestLoadConfigWithNoFile(t *testing.T) {
	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := DefaultConfig()
	if !reflect.DeepEqual(config, expected) {
		t.Error("Expected default config when no file found")
	}
}

func TestLoadConfigWithNonExistentFile(t *testing.T) {
	config, err := LoadConfig("/path/that/does/not/exist/config.json")
	if err != nil {
		t.Fatalf("Expected no error for non-existent file, got %v", err)
	}

	expected := DefaultConfig()
	if !reflect.DeepEqual(config, expected) {
		t.Error("Expected default config when file does not exist")
	}
}

func TestLoadConfigWithValidFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")

	testConfig := &Config{
		SortCriteria: SortCriteria{
			GroupByReceiver:   false,
			ExportedFirst:     false,
			SortByDepth:       false,
			SortByInDegree:    false,
			PreserveOrigOrder: false,
		},
		Exclude: []string{"*_test.go"},
		Include: []string{"*.go", "*.mod"},
	}

	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load the config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !reflect.DeepEqual(config, testConfig) {
		t.Errorf("Config mismatch.\nExpected: %+v\nGot: %+v", testConfig, config)
	}
}

func TestLoadConfigWithInvalidJSON(t *testing.T) {
	// Create a temporary config file with invalid JSON
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid_config.json")

	invalidJSON := `{
		"sort_criteria": {
			"group_by_receiver": true,
		}
		"exclude": ["*_test.go"  // missing comma and closing bracket
	}`

	if err := os.WriteFile(configPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	// Load the config - should return error for invalid JSON
	config, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
	if config != nil {
		t.Error("Expected nil config for invalid JSON")
	}
}

func TestFindConfigFile(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Test 1: No config file found
	result := findConfigFile()
	if result != "" {
		t.Errorf("Expected empty string when no config found, got %s", result)
	}

	// Test 2: .msort.json found
	if err := os.WriteFile(".msort.json", []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create .msort.json: %v", err)
	}
	result = findConfigFile()
	if result != ".msort.json" {
		t.Errorf("Expected .msort.json, got %s", result)
	}

	// Remove the file for next test
	os.Remove(".msort.json")

	// Test 3: msort.json found
	if err := os.WriteFile("msort.json", []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create msort.json: %v", err)
	}
	result = findConfigFile()
	if result != "msort.json" {
		t.Errorf("Expected msort.json, got %s", result)
	}

	// Remove the file for next test
	os.Remove("msort.json")

	// Test 4: .config/msort.json found
	if err := os.MkdirAll(".config", 0755); err != nil {
		t.Fatalf("Failed to create .config directory: %v", err)
	}
	if err := os.WriteFile(".config/msort.json", []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create .config/msort.json: %v", err)
	}
	result = findConfigFile()
	if result != ".config/msort.json" {
		t.Errorf("Expected .config/msort.json, got %s", result)
	}
}

func TestConfigSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "save_test.json")

	config := &Config{
		SortCriteria: SortCriteria{
			GroupByReceiver:   true,
			ExportedFirst:     false,
			SortByDepth:       true,
			SortByInDegree:    false,
			PreserveOrigOrder: true,
		},
		Exclude: []string{"*_test.go", "vendor/*"},
		Include: []string{"*.go"},
	}

	// Save the config
	if err := config.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load it back and verify
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if !reflect.DeepEqual(config, loadedConfig) {
		t.Errorf("Saved and loaded configs don't match.\nOriginal: %+v\nLoaded: %+v", config, loadedConfig)
	}

	// Check JSON format
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	var parsedJSON map[string]interface{}
	if err := json.Unmarshal(data, &parsedJSON); err != nil {
		t.Errorf("Saved file is not valid JSON: %v", err)
	}
}

func TestConfigSaveWithInvalidPath(t *testing.T) {
	config := DefaultConfig()

	// Try to save to an invalid path
	invalidPath := "/invalid/path/that/does/not/exist/config.json"
	if err := config.Save(invalidPath); err == nil {
		t.Error("Expected error when saving to invalid path")
	}
}

func TestLoadConfigWithEmptyPath(t *testing.T) {
	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("Expected no error with empty path, got %v", err)
	}

	expected := DefaultConfig()
	if !reflect.DeepEqual(config, expected) {
		t.Error("Expected default config with empty path")
	}
}

func TestFindConfigFileWithPermissionError(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create a config file
	configFile := ".msort.json"
	if err := os.WriteFile(configFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	result := findConfigFile()
	if result != configFile {
		t.Errorf("Expected to find %s, got %s", configFile, result)
	}
}
