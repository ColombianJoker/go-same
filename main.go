package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"hash"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/md4"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
)

// A map to store hashes and the list of files with that hash
var filesByHash = make(map[string][]string)

func getHash(algo string) (hash.Hash, error) {
	switch strings.ToLower(algo) {
	case "md4":
		return md4.New(), nil
	case "md5":
		return md5.New(), nil
	case "sha1":
		return sha1.New(), nil
	case "sha224":
		return sha256.New224(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha384":
		return sha512.New384(), nil
	case "sha512":
		return sha512.New(), nil
	case "ripemd160":
		return ripemd160.New(), nil
	case "sha3-224":
		return sha3.New224(), nil
	case "sha3-256":
		return sha3.New256(), nil
	case "sha3-384":
		return sha3.New384(), nil
	case "sha3-512":
		return sha3.New512(), nil
	case "blake2b":
		return blake2b.New512(nil)
	case "shake128":
		return sha3.NewShake128(), nil
	case "shake256":
		return sha3.NewShake256(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", algo)
	}
}

// hashFile calculates the hash of a file's content
func hashFile(filePath string, algo string) (string, error) {
	h, err := getHash(algo)
	if err != nil {
		return "", err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	// For XOFs like SHAKE, we need to read a specific output length.
	if strings.HasPrefix(strings.ToLower(algo), "shake") {
		output := make([]byte, 64)
		h.Sum(output[:0])
		return fmt.Sprintf("%x", output), nil
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// ProcessOneFile now calculates a hash and updates the map
func ProcessOneFile(filePath string, fileCount *int, verbose bool, blockSize int, rowSize int, hashAlgo string) {
	info, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Error checking file size for %s: %v", filePath, err)
		if verbose {
			fmt.Println("!")
		}
		*fileCount++
		return
	}

	// Check for zero-sized file
	if info.Size() == 0 {
		filesByHash["0-byte-file"] = append(filesByHash["0-byte-file"], filePath)
		*fileCount++
		if verbose {
			fmt.Printf(".")
		}
	} else {
		// For non-zero files, calculate the hash
		fileHash, err := hashFile(filePath, hashAlgo)
		if err != nil {
			log.Printf("Error hashing file %s: %v", filePath, err)
			return
		}

		filesByHash[fileHash] = append(filesByHash[fileHash], filePath)
		*fileCount++
		if verbose {
			fmt.Printf("=")
		}
	}

	// Now handle the progress bar display, which is only a concern for verbose output
	if verbose {
		if *fileCount%blockSize == 0 {
			if (*fileCount/blockSize)%rowSize != 0 {
				fmt.Printf(" ")
			}
			if (*fileCount/blockSize)%rowSize == 0 {
				fmt.Printf(" [%s]\n", strconv.FormatInt(int64(*fileCount), 10))
			}
		}
	}
}

func walkAndProcess(path string, fileCount *int, verbose, recursive, followLinks bool, blockSize, rowSize int, hashAlgo string) {
	walker := func(walkerPath string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", walkerPath, err)
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			if !recursive && walkerPath != path {
				return filepath.SkipDir
			}
			return nil
		}
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
					ProcessOneFile(resolvedPath, fileCount, verbose, blockSize, rowSize, hashAlgo)
				}
			}
			return nil
		}
		ProcessOneFile(walkerPath, fileCount, verbose, blockSize, rowSize, hashAlgo)
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
	var skipZeroSized bool
	var blockSize int
	var rowSize int
	var digits int
	var hashAlgo string

	// New flag for skipping zero-sized files
	flag.BoolVar(&skipZeroSized, "z", false, "Skip printing zero-sized files in the output.")
	flag.BoolVar(&skipZeroSized, "skip-zero-sized", false, "Skip printing zero-sized files in the output.")
	flag.BoolVar(&skipZeroSized, "skip-zero", false, "Skip printing zero-sized files in the output.")

	flag.StringVar(&hashAlgo, "a", "sha256", "Select the hashing algorithm.")
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
		// ... your existing list of algorithms goes here ...
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
			walkAndProcess(path, &processedFileCount, verboseFlag, recursiveFlag, followLinksFlag, blockSize, rowSize, hashAlgo)
		} else {
			ProcessOneFile(path, &processedFileCount, verboseFlag, blockSize, rowSize, hashAlgo)
		}
	}

	if verboseFlag {
		fmt.Println("\n--- Duplicate files found ---")
		foundDuplicates := false
		for h, paths := range filesByHash {
			// Skip the "0-byte-file" hash if the skip flag is set
			if h == "0-byte-file" && skipZeroSized {
				continue
			}

			if len(paths) > 1 {
				foundDuplicates = true
				fmt.Printf("%s:\n", h)
				for _, p := range paths {
					fmt.Printf("  %s\n", p)
				}
				fmt.Println()
			}
		}
		if !foundDuplicates {
			fmt.Println("No duplicate files found.")
		}
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
