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
		fmt.Fprintf(os.Stderr, "Within each run of a type's methods, with no declaration or free-standing comment between them:\n")
		fmt.Fprintf(os.Stderr, "  1. Exported methods first, in their current order\n")
		fmt.Fprintf(os.Stderr, "  2. Then the rest in call order, each followed by the helpers it uses\n")
		fmt.Fprintf(os.Stderr, "Nothing moves out of its section.\n")
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
