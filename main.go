package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

// ProcessOneFile now only updates the progress bar
func ProcessOneFile(fileCount *int, verbose bool, blockSize int, rowSize int, digits int) {
	if verbose {
		// Print a progress symbol for each file
		fmt.Printf("=")

		*fileCount++

		// Check if a block is complete
		if *fileCount%blockSize == 0 {
			// Print a space after each block, unless it's a new row
			if (*fileCount/blockSize)%rowSize != 0 {
				fmt.Printf(" ")
			}

			// Check if a row is complete
			if (*fileCount/blockSize)%rowSize == 0 {
				fmt.Printf(" [%s]\n", strconv.FormatInt(int64(*fileCount), 10))
			}
		}
	}
}

// ProcessOneDirectory is still a placeholder
func ProcessOneDirectory(dirPath string) {}

func main() {
	var verboseFlag bool
	var recursiveFlag bool
	var availableFlag bool
	var blockSize int
	var rowSize int
	var digits int

	// Define the flags
	flag.BoolVar(&availableFlag, "A", false, "List available algorithms and exit.")
	flag.BoolVar(&availableFlag, "available", false, "List available algorithms and exit.")
	flag.BoolVar(&availableFlag, "algorithms", false, "List available algorithms and exit.")

	flag.BoolVar(&recursiveFlag, "r", false, "Process directories recursively.")
	flag.BoolVar(&recursiveFlag, "recursive", false, "Process directories recursively.")

	flag.BoolVar(&verboseFlag, "v", false, "Enable verbose output.")
	flag.BoolVar(&verboseFlag, "verbose", false, "Enable verbose output.")

	flag.IntVar(&blockSize, "blockSize", 10, "Number of '=' symbols per block.")
	flag.IntVar(&rowSize, "rowSize", 10, "Number of blocks per row.")
	flag.IntVar(&digits, "digits", 6, "Number of digits for the file counter.")

	// Parse the flags
	flag.Parse()

	// Handle the -A/--available flag first, as it's a special case
	if availableFlag {
		fmt.Println("Available Hashing Algorithms:")
		// ... your existing list of algorithms goes here ...
		return
	}

	// Get the paths from the command-line arguments
	paths := flag.Args()

	if len(paths) == 0 {
		fmt.Println("Usage: same [options] <path1> <path2> ...")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// First pass: Count files for a more accurate progress bar (optional, but good practice)
	// This example skips the counting to keep the code simpler.

	// Second pass: Process files and display progress
	var processedFileCount int
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			continue
		}

		if info.IsDir() {
			ProcessOneDirectory(path)
			if recursiveFlag {
				filepath.Walk(path, func(walkerPath string, walkerInfo os.FileInfo, err error) error {
					if err != nil {
						log.Printf("Error accessing path %s: %v", walkerPath, err)
						return nil
					}
					if !walkerInfo.IsDir() {
						ProcessOneFile(&processedFileCount, verboseFlag, blockSize, rowSize, digits)
					}
					return nil
				})
			}
		} else { // It's a file
			ProcessOneFile(&processedFileCount, verboseFlag, blockSize, rowSize, digits)
		}
	}
	fmt.Println("\nCompleted.")
}
