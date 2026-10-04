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
		fmt.Fprintf(os.Stderr, "Within each type, methods read top-down:\n")
		fmt.Fprintf(os.Stderr, "  1. Entry points first, exported ones before the rest\n")
		fmt.Fprintf(os.Stderr, "  2. Each followed by the helpers it uses, in the order it uses them\n")
		fmt.Fprintf(os.Stderr, "  3. A helper several methods use follows the first of them\n")
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
