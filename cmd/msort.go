package cmd

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/borovikovd/go-msort/pkg/sorter"
)

type Config struct {
	DryRun    bool
	Recursive bool
	Verbose   bool
	Paths     []string
}

func Run(config *Config) error {
	for _, path := range config.Paths {
		if err := processPath(path, config); err != nil {
			return fmt.Errorf("processing %s: %w", path, err)
		}
	}
	return nil
}

func processPath(path string, config *Config) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return processDirectory(path, config)
	}

	if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
		return processFile(path, config)
	}

	return nil
}

func processDirectory(dir string, config *Config) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			if config.Recursive && !strings.HasPrefix(entry.Name(), ".") {
				if err := processDirectory(path, config); err != nil {
					return err
				}
			}
			continue
		}

		if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			if err := processFile(path, config); err != nil {
				return err
			}
		}
	}

	return nil
}

func processFile(filename string, config *Config) error {
	if config.Verbose {
		fmt.Printf("Processing: %s\n", filename)
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", filename, err)
	}

	methodSorter := sorter.New(fset, node)
	sorted, changed, err := methodSorter.Sort()
	if err != nil {
		return fmt.Errorf("sorting methods in %s: %w", filename, err)
	}

	if !changed {
		if config.Verbose {
			fmt.Printf("  No changes needed\n")
		}
		return nil
	}

	if config.DryRun {
		fmt.Printf("Would sort methods in: %s\n", filename)
		return nil
	}

	if err := sorter.WriteFile(filename, sorted); err != nil {
		return fmt.Errorf("writing sorted file %s: %w", filename, err)
	}

	if config.Verbose {
		fmt.Printf("  Methods sorted\n")
	}

	return nil
}
