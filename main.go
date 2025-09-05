package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

// ProcessOneFile processes a single file, printing a '.' if it's 0 bytes or '=' otherwise.
func ProcessOneFile(filePath string, fileCount *int, verbose bool, blockSize int, rowSize int) {
	if verbose {
		// Get file info using the provided fully-qualified path
		info, err := os.Stat(filePath)
		if err != nil {
			log.Printf("Error checking file size for %s: %v", filePath, err)
			return
		}

		// Print '.' for zero-sized files, '=' for others
		if info.Size() == 0 {
			fmt.Printf(".")
		} else {
			fmt.Printf("=")
		}

		*fileCount++

		// Check if a block is complete
		if *fileCount%blockSize == 0 {
			// Print a space after each block, unless it's a new row
			if (*fileCount/blockSize)%rowSize != 0 {
				fmt.Printf(" ")
			}

			// Check if a row is complete
			if (*fileCount/blockSize)%rowSize == 0 {
				fmt.Printf("[%s]\n", strconv.FormatInt(int64(*fileCount), 10))
			}
		}
	}
}
func main() {
	var verboseFlag bool
	var recursiveFlag bool
	var availableFlag bool
	var followLinksFlag bool
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

	flag.BoolVar(&followLinksFlag, "l", false, "Follow symbolic links.")
	flag.BoolVar(&followLinksFlag, "follow-links", false, "Follow symbolic links.")

	flag.IntVar(&blockSize, "blockSize", 10, "Number of '=' symbols per block.")
	flag.IntVar(&rowSize, "rowSize", 10, "Number of blocks per row.")
	flag.IntVar(&digits, "digits", 6, "Number of digits for the file counter.")

	// Parse the flags
	flag.Parse()

	if availableFlag {
		fmt.Println("Available Hashing Algorithms:")
		// ... your existing list of algorithms goes here ...
		return
	}

	paths := flag.Args()
	if len(paths) == 0 {
		fmt.Println("Usage: same [options] <path1> <path2> ...")
		flag.PrintDefaults()
		os.Exit(1)
	}

	var processedFileCount int
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			if followLinksFlag {
				resolvedPath, err := filepath.EvalSymlinks(path)
				if err != nil {
					log.Printf("Error resolving symlink %s: %v", path, err)
					continue
				}
				path = resolvedPath
				info, err = os.Stat(path)
				if err != nil {
					log.Printf("Error accessing resolved path %s: %v", path, err)
					continue
				}
			} else {
				continue
			}
		}

		if info.IsDir() {
			if recursiveFlag {
				filepath.Walk(path, func(walkerPath string, walkerInfo os.FileInfo, err error) error {
					if err != nil {
						log.Printf("Error accessing path %s: %v", walkerPath, err)
						return nil
					}

					linkInfo, linkErr := os.Lstat(walkerPath)
					if linkErr != nil {
						log.Printf("Error checking link %s: %v", walkerPath, linkErr)
						return nil
					}

					if linkInfo.Mode()&os.ModeSymlink != 0 {
						if followLinksFlag {
							resolvedPath, err := filepath.EvalSymlinks(walkerPath)
							if err != nil {
								log.Printf("Error following symlink %s: %v", walkerPath, err)
								return nil
							}

							resolvedInfo, err := os.Stat(resolvedPath)
							if err != nil {
								log.Printf("Error statting resolved path %s: %v", resolvedPath, err)
								return nil
							}
							if !resolvedInfo.IsDir() {
								ProcessOneFile(resolvedPath, &processedFileCount, verboseFlag, blockSize, rowSize)
							}
						}
					} else if !walkerInfo.IsDir() {
						ProcessOneFile(walkerPath, &processedFileCount, verboseFlag, blockSize, rowSize)
					}
					return nil
				})
			}
		} else {
			ProcessOneFile(path, &processedFileCount, verboseFlag, blockSize, rowSize)
		}
	}
	if verboseFlag {
		fmt.Println("\nCompleted.")
	}
}
