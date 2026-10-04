package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/borovikovd/gomsort/pkg/sorter"
)

type Config struct {
	DryRun  bool
	Verbose bool
	Paths   []string
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

	if isSortable(path) {
		return processFile(path, config)
	}

	return nil
}

// processDirectory sorts every Go file under dir. A file it can't sort is
// reported on stderr and the others are still sorted, as gofmt does; it then
// returns an error saying how many failed.
func processDirectory(dir string, config *Config) error {
	failed := 0
	if err := sortTree(dir, config, &failed); err != nil {
		return err
	}
	if failed > 0 {
		return fmt.Errorf("%d file(s) couldn't be sorted", failed)
	}
	return nil
}

// sortTree sorts the Go files under dir in walk order, reporting each one
// it can't sort on stderr and counting it in failed.
func sortTree(dir string, config *Config, failed *int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		switch {
		case entry.IsDir() && !skipDir(entry.Name()):
			if err := sortTree(path, config, failed); err != nil {
				return err
			}
		case !entry.IsDir() && isSortable(entry.Name()):
			if err := processFile(path, config); err != nil {
				fmt.Fprintln(os.Stderr, err)
				*failed++
			}
		}
	}
	return nil
}

func isSortable(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// skipDir reports whether a directory is one the go command ignores:
// testdata, vendor, and names starting with "." or "_".
func skipDir(name string) bool {
	return name == "testdata" || name == "vendor" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

func processFile(filename string, config *Config) error {
	if config.Verbose {
		fmt.Printf("Processing: %s\n", filename)
	}

	source, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filename, err)
	}

	methodSorter, err := sorter.NewFromSource(string(source))
	if err != nil {
		return fmt.Errorf("parsing %s: %w", filename, err)
	}
	sorted, changed, err := methodSorter.Sort()
	if err != nil {
		return fmt.Errorf("sorting methods in %s: %w", filename, err)
	}

	switch {
	case !changed:
		if config.Verbose {
			fmt.Printf("  No changes needed\n")
		}
	case config.DryRun:
		fmt.Printf("Would sort methods in: %s\n", filename)
	default:
		if err := sorter.WriteFile(filename, sorted); err != nil {
			return fmt.Errorf("writing sorted file %s: %w", filename, err)
		}
		if config.Verbose {
			fmt.Printf("  Methods sorted\n")
		}
	}
	return nil
}
