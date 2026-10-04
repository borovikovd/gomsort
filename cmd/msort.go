package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

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
		// Check if we're in a Go module context when processing directories
		if err := checkGoModule(path); err != nil {
			return err
		}
		return processDirectory(path, config)
	}

	if isSortable(path) {
		return processFile(path, config)
	}

	return nil
}

func checkGoModule(dir string) error {
	// Look for go.mod in current directory or any parent directory
	current := dir
	for {
		goModPath := filepath.Join(current, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return nil // Found go.mod
		}

		parent := filepath.Dir(current)
		if parent == current {
			break // Reached filesystem root
		}
		current = parent
	}

	return fmt.Errorf("go.mod file not found in current directory or any parent directory; see 'go help modules'")
}

func processDirectory(dir string, config *Config) error {
	files, err := goFiles(dir)
	if err != nil {
		return err
	}
	return processFiles(files, config)
}

// goFiles lists the files under dir that gomsort sorts, in walk order.
func goFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		switch {
		case entry.IsDir() && !skipDir(entry.Name()):
			sub, err := goFiles(path)
			if err != nil {
				return nil, err
			}
			files = append(files, sub...)
		case !entry.IsDir() && isSortable(entry.Name()):
			files = append(files, path)
		}
	}
	return files, nil
}

func isSortable(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// processFiles sorts files in parallel, one per CPU at a time, so memory
// follows the largest files being sorted at once rather than how many there
// are. It prints what happened in file order, and returns the first error in
// file order.
func processFiles(files []string, config *Config) error {
	type result struct {
		output string
		err    error
	}
	results := make([]result, len(files))
	next := make(chan int)
	var wg sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), len(files)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				results[i].output, results[i].err = sortFile(files[i], config)
			}
		}()
	}
	for i := range files {
		next <- i
	}
	close(next)
	wg.Wait()

	for _, r := range results {
		fmt.Print(r.output)
		if r.err != nil {
			return r.err
		}
	}
	return nil
}

// skipDir reports whether a directory is one the go command ignores:
// testdata, vendor, and names starting with "." or "_".
func skipDir(name string) bool {
	return name == "testdata" || name == "vendor" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

func processFile(filename string, config *Config) error {
	output, err := sortFile(filename, config)
	fmt.Print(output)
	return err
}

// sortFile sorts one file, writing it unless it's a dry run, and returns
// what to print about it.
func sortFile(filename string, config *Config) (string, error) {
	var out strings.Builder
	if config.Verbose {
		fmt.Fprintf(&out, "Processing: %s\n", filename)
	}

	source, err := os.ReadFile(filename)
	if err != nil {
		return out.String(), fmt.Errorf("reading %s: %w", filename, err)
	}

	methodSorter, err := sorter.NewFromSource(string(source))
	if err != nil {
		return out.String(), fmt.Errorf("parsing %s: %w", filename, err)
	}
	sorted, changed, err := methodSorter.Sort()
	if err != nil {
		return out.String(), fmt.Errorf("sorting methods in %s: %w", filename, err)
	}

	switch {
	case !changed:
		if config.Verbose {
			out.WriteString("  No changes needed\n")
		}
	case config.DryRun:
		fmt.Fprintf(&out, "Would sort methods in: %s\n", filename)
	default:
		if err := sorter.WriteFile(filename, sorted); err != nil {
			return out.String(), fmt.Errorf("writing sorted file %s: %w", filename, err)
		}
		if config.Verbose {
			out.WriteString("  Methods sorted\n")
		}
	}
	return out.String(), nil
}
