package main

import (
	"fmt"
	"flag"
	"os"
)

func main() {
	var verboseFlag bool
	var recursiveFlag bool
	var availableFlag bool

	// Define the flags and their short/long forms
	flag.BoolVar(&availableFlag, "A", false, "List available algorithms and exit.")
	flag.BoolVar(&availableFlag, "available", false, "List available algorithms and exit.")
	flag.BoolVar(&availableFlag, "algorithms", false, "List available algorithms and exit.")

	flag.BoolVar(&recursiveFlag, "r", false, "Process directories recursively.")
	flag.BoolVar(&recursiveFlag, "recursive", false, "Process directories recursively.")

	flag.BoolVar(&verboseFlag, "v", false, "Enable verbose output.")
	flag.BoolVar(&verboseFlag, "verbose", false, "Enable verbose output.")

	// Change the default help behavior to show both short and long forms
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s [options] <path1> <path2> ...\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	// Parse the flags
	flag.Parse()

	// Handle the -A/--available flag first, as it's a special case that exits
	if availableFlag {
		fmt.Println("Available Hashing Algorithms:")
		fmt.Println("  - MD4")
		fmt.Println("  - MD5")
		fmt.Println("  - SHA-1")
		fmt.Println("  - SHA-2 (SHA-224, SHA-256, SHA-384, SHA-512)")
		// End the program after listing
		return
	}

	// Handle the -v/--verbose flag
	if verboseFlag {
		fmt.Println("Verbose mode enabled.")
	}

	// Handle the -r/--recursive flag
	if recursiveFlag {
		fmt.Println("Recursive mode enabled.")
	}

	// Example of processing paths. The flag.Args() slice contains all non-flag arguments.
	if len(flag.Args()) == 0 {
		fmt.Println("No paths provided. Use -h or --help for usage information.")
	} else {
		fmt.Println("Processing paths:", flag.Args())
	}
}
