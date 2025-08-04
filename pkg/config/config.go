package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	SortCriteria SortCriteria `json:"sort_criteria"`
	Exclude      []string     `json:"exclude"`
	Include      []string     `json:"include"`
}

type SortCriteria struct {
	GroupByReceiver   bool `json:"group_by_receiver"`
	ExportedFirst     bool `json:"exported_first"`
	SortByDepth       bool `json:"sort_by_depth"`
	SortByInDegree    bool `json:"sort_by_in_degree"`
	PreserveOrigOrder bool `json:"preserve_original_order"`
}

func DefaultConfig() *Config {
	return &Config{
		SortCriteria: SortCriteria{
			GroupByReceiver:   true,
			ExportedFirst:     true,
			SortByDepth:       true,
			SortByInDegree:    true,
			PreserveOrigOrder: true,
		},
		Exclude: []string{},
		Include: []string{"*.go"},
	}
}

func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = findConfigFile()
	}

	if configPath == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultConfig(), nil
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func findConfigFile() string {
	candidates := []string{
		".msort.json",
		"msort.json",
		".config/msort.json",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	homeConfig := filepath.Join(home, ".config", "msort", "config.json")
	if _, err := os.Stat(homeConfig); err == nil {
		return homeConfig
	}

	return ""
}

func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
