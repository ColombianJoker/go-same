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
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/md4"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
	"golang.org/x/sys/unix"
)

var filesByHash = make(map[string][]string)

const xattrName = "user.file_hash"

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
	// We'll use a fixed length of 64 bytes for consistent output.
	if strings.HasPrefix(strings.ToLower(algo), "shake") {
		output := make([]byte, 64)
		h.Sum(output[:0])
		return fmt.Sprintf("%x", output), nil
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func setXattrHash(filePath string, hashValue string) {
	err := unix.Setxattr(filePath, xattrName, []byte(hashValue), 0)
	if err != nil {
		log.Printf("Warning: Could not set extended attribute for %s. Reason: %v", filePath, err)
	}
}

func main() {
	var hashAlgo = flag.String("a", "sha256", "Select the hashing algorithm (md4, md5, sha1, ripemd160, sha224, sha256, sha384, sha512, sha3-224, sha3-256, sha3-384, sha3-512, blake2b, shake128, shake256).")
	var setXattr = flag.Bool("x", false, "Store the calculated hash as an extended attribute on each file.")
	var availableFlag bool
	flag.BoolVar(&availableFlag, "A", false, "List available algorithms and exit.")
	flag.BoolVar(&availableFlag, "available", false, "List available algorithms and exit.")
	flag.BoolVar(&availableFlag, "algorithms", false, "List available algorithms and exit.")
	flag.Parse()

	if availableFlag {
		fmt.Println("Available Hashing Algorithms:")
		fmt.Println("  - MD4")
		fmt.Println("  - MD5")
		fmt.Println("  - SHA-1")
		fmt.Println("  - RIPEMD-160")
		fmt.Println("  - SHA-2 (sha224, sha256, sha384, sha512)")
		fmt.Println("  - SHA-3 (sha3-224, sha3-256, sha3-384, sha3-512)")
		fmt.Println("  - SHAKE (shake128, shake256) - eXtendable-Output Functions")
		fmt.Println("  - BLAKE2b")
		return
	}

	if len(flag.Args()) == 0 {
		fmt.Println("Usage: same [options] <path1> <path2> ...")
		flag.PrintDefaults()
		os.Exit(1)
	}

	for _, p := range flag.Args() {
		filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("Error accessing path %s: %v", path, err)
				return nil
			}
			if !info.IsDir() {
				hashVal, err := hashFile(path, *hashAlgo)
				if err != nil {
					log.Printf("Error hashing file %s: %v", path, err)
					return nil
				}
				filesByHash[hashVal] = append(filesByHash[hashVal], path)
				if *setXattr {
					setXattrHash(path, hashVal)
				}
			}
			return nil
		})
	}

	fmt.Println("--- Duplicate files found ---")
	foundDuplicates := false
	for h, paths := range filesByHash {
		if len(paths) > 1 {
			foundDuplicates = true
			fmt.Printf("Hash: %s\n", h)
			for _, p := range paths {
				fmt.Printf("  - %s\n", p)
			}
			fmt.Println()
		}
	}

	if !foundDuplicates {
		fmt.Println("No duplicate files found.")
	}
}
