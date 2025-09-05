package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func ProcessOneFile(filePath string, fileCount *int, verbose bool, blockSize int, rowSize int) {
	if verbose {
		info, err := os.Stat(filePath)
		if err != nil {
			log.Printf("Error checking file size for %s: %v", filePath, err)
			fmt.Printf("!")
			return
		}

		if info.Size() == 0 {
			fmt.Printf(".")
		} else {
			fmt.Printf("=")
		}

		*fileCount++

		if *fileCount%blockSize == 0 {
			if (*fileCount/blockSize)%rowSize != 0 {
				fmt.Printf(" ")
			}
			if (*fileCount/blockSize)%rowSize == 0 {
				fmt.Printf(" [%s]\n", strconv.FormatInt(int64(*fileCount), 10))
			}
		}
	} else {
		*fileCount++
	}
}

// walkAndProcess recursively walks a path and processes files.
func walkAndProcess(path string, fileCount *int, verbose, recursive, followLinks bool, blockSize, rowSize int) {
	walker := func(walkerPath string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", walkerPath, err)
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		// Skip directories unless recursive flag is off
		if info.IsDir() {
			if !recursive && walkerPath != path {
				return filepath.SkipDir
			}
			return nil
		}

		// Handle symbolic links
		if info.Mode()&os.ModeSymlink != 0 {
			if followLinks {
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
					ProcessOneFile(resolvedPath, fileCount, verbose, blockSize, rowSize)
				}
			}
			return nil
		}

		ProcessOneFile(walkerPath, fileCount, verbose, blockSize, rowSize)
		return nil
	}

	filepath.WalkDir(path, walker)
}

func main() {
	var verboseFlag bool
	var recursiveFlag bool
	var availableFlag bool
	var followLinksFlag bool
	var showTimeFlag bool
	var blockSize int
	var rowSize int
	var digits int

	flag.BoolVar(&availableFlag, "A", false, "List available algorithms and exit.")
	flag.BoolVar(&availableFlag, "available", false, "List available algorithms and exit.")
	flag.BoolVar(&availableFlag, "algorithms", false, "List available algorithms and exit.")
	flag.BoolVar(&recursiveFlag, "r", false, "Process directories recursively.")
	flag.BoolVar(&recursiveFlag, "recursive", false, "Process directories recursively.")
	flag.BoolVar(&verboseFlag, "v", false, "Enable verbose output.")
	flag.BoolVar(&verboseFlag, "verbose", false, "Enable verbose output.")
	flag.BoolVar(&followLinksFlag, "l", false, "Follow symbolic links.")
	flag.BoolVar(&followLinksFlag, "follow-links", false, "Follow symbolic links.")
	flag.BoolVar(&showTimeFlag, "t", false, "Show time taken to process.")
	flag.BoolVar(&showTimeFlag, "show-time", false, "Show time taken to process.")

	flag.IntVar(&blockSize, "blockSize", 10, "Number of '=' symbols per block.")
	flag.IntVar(&rowSize, "rowSize", 10, "Number of blocks per row.")
	flag.IntVar(&digits, "digits", 6, "Number of digits for the file counter.")

	flag.Parse()

	if availableFlag {
		fmt.Println("Available Hashing Algorithms:")
		return
	}

	var startTime time.Time
	if showTimeFlag {
		startTime = time.Now()
		if verboseFlag {
			fmt.Printf("Start time: %s\n", startTime.Format("2006-01-02 15:04:05"))
		}
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
			walkAndProcess(path, &processedFileCount, verboseFlag, recursiveFlag, followLinksFlag, blockSize, rowSize)
		} else {
			ProcessOneFile(path, &processedFileCount, verboseFlag, blockSize, rowSize)
		}
	}

	if verboseFlag {
		fmt.Println()
	}

	if showTimeFlag {
		endTime := time.Now()
		elapsed := endTime.Sub(startTime)

		if verboseFlag {
			fmt.Printf("End time: %s\n", endTime.Format("2006-01-02 15:04:05"))
		}

		if processedFileCount > 0 {
			fmt.Printf("%d files, %.4f seconds / file\n", processedFileCount, elapsed.Seconds()/float64(processedFileCount))
		} else {
			fmt.Printf("No files found.\n")
		}
	}
}
