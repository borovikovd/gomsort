package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/borovikovd/gomsort/cmd"
)

func main() {
	var (
		dryRun  = flag.Bool("n", false, "dry run - show what would be changed without modifying files")
		verbose = flag.Bool("v", false, "verbose output")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [files/directories...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\ngo-msort sorts Go methods within types for better readability.\n")
		fmt.Fprintf(os.Stderr, "Recursively processes directories like 'go fmt'.\n")
		fmt.Fprintf(os.Stderr, "Methods are sorted by:\n")
		fmt.Fprintf(os.Stderr, "  1. Receiver type (grouped together)\n")
		fmt.Fprintf(os.Stderr, "  2. Exported methods first\n")
		fmt.Fprintf(os.Stderr, "  3. Entry points (low call depth) first\n")
		fmt.Fprintf(os.Stderr, "  4. Helper methods (high in-degree) last\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"."}
	}

	config := &cmd.Config{
		DryRun:  *dryRun,
		Verbose: *verbose,
		Paths:   args,
	}

	if err := cmd.Run(config); err != nil {
		log.Fatal(err)
	}
}
