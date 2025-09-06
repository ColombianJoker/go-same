package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/md4"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"

	"github.com/pkg/xattr"
)

// A map to store hashes and the list of files with that hash
var filesByHash = make(map[string][]string)

// Flag messages as constants for better maintainability
const (
	hashAlgoUsage      = "Select the hashing algorithm."
	availableUsage     = "List available algorithms and exit."
	recursiveUsage     = "Process directories recursively."
	verboseUsage       = "Enable verbose output."
	followLinksUsage   = "Follow symbolic links."
	showTimeUsage      = "Show time taken to process."
	skipZeroSizedUsage = "Skip printing zero-sized files in the output."
	storeXattrUsage    = "Check for and store hash in extended attributes."
	recreateXattrUsage = "Always recalculate and store hash in extended attributes."
	blockSizeUsage     = "Number of '=' symbols per block."
	rowSizeUsage       = "Number of blocks per row."
	digitsUsage        = "Number of digits for the file counter."
	duplicatesUsage    = "Only show duplicate files (with the same hash) according to specified rules."
	debugUsage         = "Enable debug mode (disables other output and prints hash, file name per file)."
	noShowHashesUsage  = "Don't show the hashes when listing duplicates."
	noDotUsage         = "Cut `./` from the start of names of files when listing."
	stderrUsage        = "Send progress bars to stderr."
)

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
	if strings.HasPrefix(strings.ToLower(algo), "shake") {
		output := make([]byte, 64)
		h.Sum(output[:0])
		return fmt.Sprintf("%x", output), nil
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// ProcessOneFile now calculates a hash and updates the map
func ProcessOneFile(filePath string, fileCount *int, verbose, storeXattr, recreateXattr, debugFlag, stderrProgress bool, blockSize, rowSize int, hashAlgo string) {
	info, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Error checking file size for %s: %v", filePath, err)
		if verbose && !stderrProgress {
			fmt.Println("!")
		} else if verbose && stderrProgress {
			fmt.Fprintln(os.Stderr, "!")
		}
		*fileCount++
		return
	}

	// For zero-sized files, we can just use a constant hash
	if info.Size() == 0 {
		filesByHash["0-byte-file"] = append(filesByHash["0-byte-file"], filePath)
		*fileCount++
		if verbose && !stderrProgress {
			fmt.Printf(".")
		} else if verbose && stderrProgress {
			fmt.Fprintf(os.Stderr, ".")
		}
		return
	}

	var fileHash string
	xattrName := "user.same-hash." + hashAlgo

	// Logic for extended attributes
	if storeXattr || recreateXattr {
		if storeXattr && !recreateXattr {
			// Try to get hash from extended attribute
			xattrValue, err := xattr.Get(filePath, xattrName)
			if err == nil {
				fileHash = string(xattrValue)
			} else {
				// Attribute doesn't exist or error, calculate and store it
				fileHash, err = hashFile(filePath, hashAlgo)
				if err != nil {
					log.Printf("Error hashing file %s: %v", filePath, err)
					return
				}
				if err := xattr.Set(filePath, xattrName, []byte(fileHash)); err != nil {
					log.Printf("Error writing xattr for %s: %v", filePath, err)
				}
			}
		} else if recreateXattr {
			// Always recalculate and overwrite
			fileHash, err = hashFile(filePath, hashAlgo)
			if err != nil {
				log.Printf("Error hashing file %s: %v", filePath, err)
				return
			}
			if err := xattr.Set(filePath, xattrName, []byte(fileHash)); err != nil {
				log.Printf("Error writing xattr for %s: %v", filePath, err)
			}
		}
	} else {
		// No xattr option, just calculate the hash
		fileHash, err = hashFile(filePath, hashAlgo)
		if err != nil {
			log.Printf("Error hashing file %s: %v", filePath, err)
			return
		}
	}

	filesByHash[fileHash] = append(filesByHash[fileHash], filePath)
	*fileCount++

	if verbose {
		if !stderrProgress {
			fmt.Printf("=")
		} else {
			fmt.Fprintf(os.Stderr, "=")
		}
	}
	if debugFlag {
		if !stderrProgress {
			fmt.Printf("%s %s\n", fileHash, filePath)
		} else {
			fmt.Fprintf(os.Stderr, "%s %s\n", fileHash, filePath)
		}
	}

	if verbose {
		if *fileCount%blockSize == 0 {
			if (*fileCount/blockSize)%rowSize != 0 {
				if !stderrProgress {
					fmt.Printf(" ")
				} else {
					fmt.Fprintf(os.Stderr, " ")
				}
			}
			if (*fileCount/blockSize)%rowSize == 0 {
				if !stderrProgress {
					fmt.Printf(" [%6s]\n", strconv.FormatInt(int64(*fileCount), 10))
				} else {
					fmt.Fprintf(os.Stderr, " [%6s]\n", strconv.FormatInt(int64(*fileCount), 10))
				}
			}
		}
	}
}

func walkAndProcess(path string, fileCount *int, verbose, recursive, followLinks, storeXattr, recreateXattr, debugFlag, stderrProgress bool, blockSize, rowSize int, hashAlgo string) {
	walker := func(walkerPath string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", walkerPath, err)
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}

		// Skip macOS system files
		if info.Name() == ".DS_Store" || info.Name() == "Icon\r" {
			return nil
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
					ProcessOneFile(resolvedPath, fileCount, verbose, storeXattr, recreateXattr, debugFlag, stderrProgress, blockSize, rowSize, hashAlgo)
				}
			}
			return nil
		}
		ProcessOneFile(walkerPath, fileCount, verbose, storeXattr, recreateXattr, debugFlag, stderrProgress, blockSize, rowSize, hashAlgo)
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
	var duplicatesFlag bool
	var debugFlag bool
	var storeXattr bool
	var recreateXattr bool
	var blockSize int
	var rowSize int
	var digits int
	var hashAlgo string
	var noShowHashes bool
	var noDot bool
	var stderrProgress bool

	flag.StringVar(&hashAlgo, "a", "sha512", hashAlgoUsage)
	flag.BoolVar(&availableFlag, "A", false, availableUsage)
	flag.BoolVar(&availableFlag, "available", false, availableUsage)
	flag.BoolVar(&availableFlag, "algorithms", false, availableUsage)
	flag.BoolVar(&recursiveFlag, "r", false, recursiveUsage)
	flag.BoolVar(&recursiveFlag, "recursive", false, recursiveUsage)
	flag.BoolVar(&verboseFlag, "v", false, verboseUsage)
	flag.BoolVar(&verboseFlag, "verbose", false, verboseUsage)
	flag.BoolVar(&followLinksFlag, "l", false, followLinksUsage)
	flag.BoolVar(&followLinksFlag, "follow-links", false, followLinksUsage)
	flag.BoolVar(&showTimeFlag, "t", false, showTimeUsage)
	flag.BoolVar(&showTimeFlag, "show-time", false, showTimeUsage)
	flag.BoolVar(&skipZeroSized, "z", false, skipZeroSizedUsage)
	flag.BoolVar(&skipZeroSized, "skip-zero-sized", false, skipZeroSizedUsage)
	flag.BoolVar(&skipZeroSized, "skip-zero", false, skipZeroSizedUsage)
	flag.BoolVar(&duplicatesFlag, "d", false, duplicatesUsage)
	flag.BoolVar(&duplicatesFlag, "duplicates", false, duplicatesUsage)
	flag.BoolVar(&debugFlag, "DEBUG", false, debugUsage)
	flag.BoolVar(&storeXattr, "X", false, storeXattrUsage)
	flag.BoolVar(&storeXattr, "store-xattr", false, storeXattrUsage)
	flag.BoolVar(&recreateXattr, "Y", false, recreateXattrUsage)
	flag.BoolVar(&recreateXattr, "always-recreate-xattr", false, recreateXattrUsage)
	flag.IntVar(&blockSize, "blockSize", 10, blockSizeUsage)
	flag.IntVar(&rowSize, "rowSize", 10, rowSizeUsage)
	flag.IntVar(&digits, "digits", 6, digitsUsage)
	flag.BoolVar(&noShowHashes, "n", false, noShowHashesUsage)
	flag.BoolVar(&noShowHashes, "no-show", false, noShowHashesUsage)
	flag.BoolVar(&noShowHashes, "noshow", false, noShowHashesUsage)
	flag.BoolVar(&noDot, "N", false, noDotUsage)
	flag.BoolVar(&noDot, "no-dot", false, noDotUsage)
	flag.BoolVar(&noDot, "nodot", false, noDotUsage)
	flag.BoolVar(&stderrProgress, "stderr-progress", false, stderrUsage)
	flag.BoolVar(&stderrProgress, "stderr", false, stderrUsage)

	flag.Parse()

	// If debug mode is enabled, override other output flags
	if debugFlag {
		verboseFlag = false
		showTimeFlag = false
		duplicatesFlag = false
		if !stderrProgress {
			fmt.Printf("Using hashing algorithm: %s\n", hashAlgo)
		} else {
			fmt.Fprintf(os.Stderr, "Using hashing algorithm: %s\n", hashAlgo)
		}
	}

	if availableFlag {
		fmt.Println("Available Hashing Algorithms:")
		fmt.Println("- md4")
		fmt.Println("- md5")
		fmt.Println("- sha1")
		fmt.Println("- sha224")
		fmt.Println("- sha256")
		fmt.Println("- sha384")
		fmt.Println("- sha512 (default)")
		fmt.Println("- ripemd160")
		fmt.Println("- sha3-224")
		fmt.Println("- sha3-256")
		fmt.Println("- sha3-384")
		fmt.Println("- sha3-512")
		fmt.Println("- blake2b")
		fmt.Println("- shake128")
		fmt.Println("- shake256")
		return
	}

	var startTime time.Time
	if showTimeFlag {
		startTime = time.Now()
		if verboseFlag {
			if !stderrProgress {
				fmt.Printf("Start time: %s\n", startTime.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Fprintf(os.Stderr, "Start time: %s\n", startTime.Format("2006-01-02 15:04:05"))
			}
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
			walkAndProcess(path, &processedFileCount, verboseFlag, recursiveFlag, followLinksFlag, storeXattr, recreateXattr, debugFlag, stderrProgress, blockSize, rowSize, hashAlgo)
		} else {
			ProcessOneFile(path, &processedFileCount, verboseFlag, storeXattr, recreateXattr, debugFlag, stderrProgress, blockSize, rowSize, hashAlgo)
		}
	}

	if duplicatesFlag {
		if !stderrProgress {
			fmt.Println("\n--- Duplicate files found ---")
		}
		foundDuplicates := false
		for h, paths := range filesByHash {
			if h == "0-byte-file" && skipZeroSized {
				continue
			}
			if len(paths) > 1 {
				foundDuplicates = true
				if !noShowHashes {
					fmt.Printf("%s:\n", h)
				}
				sort.Slice(paths, func(i, j int) bool {
					lenI := len(paths[i])
					lenJ := len(paths[j])
					if lenI != lenJ {
						return lenI < lenJ
					}
					baseI := filepath.Base(paths[i])
					baseJ := filepath.Base(paths[j])
					return len(baseI) < len(baseJ)
				})
				// Check for files with the same shortest path and basename length
				shortestPaths := []string{}
				if len(paths) > 0 {
					shortestPathLength := len(paths[0])
					shortestBasenameLength := len(filepath.Base(paths[0]))
					for _, p := range paths {
						if len(p) == shortestPathLength && len(filepath.Base(p)) == shortestBasenameLength {
							shortestPaths = append(shortestPaths, p)
						}
					}
				}

				for _, p := range paths {
					if noDot {
						p = strings.TrimPrefix(p, "./")
					}
					isShortest := false
					for _, sp := range shortestPaths {
						if p == sp {
							isShortest = true
							break
						}
					}
					if len(shortestPaths) > 1 && isShortest {
						fmt.Printf("  %s ×\n", p)
					} else if len(shortestPaths) == 1 && isShortest {
						continue // Skip the single shortest file
					} else {
						fmt.Printf("  %s\n", p)
					}
				}
				fmt.Println()
			}
		}

		if !foundDuplicates {
			fmt.Println("No duplicate files found.")
		}
	} else if verboseFlag { // Fallback to old behavior if -d is not used but -v is
		if !stderrProgress {
			fmt.Println("\n--- Duplicate files found ---")
		}
		foundDuplicates := false
		for h, paths := range filesByHash {
			if h == "0-byte-file" && skipZeroSized {
				continue
			}
			if len(paths) > 1 {
				foundDuplicates = true
				if !noShowHashes {
					fmt.Printf("%s:\n", h)
				}
				for _, p := range paths {
					if noDot {
						p = strings.TrimPrefix(p, "./")
					}
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
			if !stderrProgress {
				fmt.Printf("End time: %s\n", endTime.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Fprintf(os.Stderr, "End time: %s\n", endTime.Format("2006-01-02 15:04:05"))
			}
		}
		if processedFileCount > 0 {
			if !stderrProgress {
				fmt.Printf("%d files, %.4f seconds / file\n", processedFileCount, elapsed.Seconds()/float64(processedFileCount))
			} else {
				fmt.Fprintf(os.Stderr, "%d files, %.4f seconds / file\n", processedFileCount, elapsed.Seconds()/float64(processedFileCount))
			}
		} else {
			if !stderrProgress {
				fmt.Printf("No files found.\n")
			} else {
				fmt.Fprintf(os.Stderr, "No files found.\n")
			}
		}
	}
}
