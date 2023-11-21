package main

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"

	"crypto/sha256"

	"github.com/bloom42/stdx/cobra"
	"github.com/zeebo/blake3"
	"golang.org/x/crypto/blake2b"
)

const (
	version = "0.1.0"

	AlgorithmSha256  = "sha256"
	AlgorithmSha512  = "sha512"
	AlgorithmBlake3  = "blake3"
	AlgorithmBlake2b = "blake2b"
)

var (
	algorithmArg string
	stdinArg     bool
)

func init() {
	rootCmd.Flags().StringVarP(&algorithmArg, "algorithm", "a", AlgorithmSha256, "Algorithm to use. Valid values are [sha256, sha512, blake3, blake2b]")
	rootCmd.Flags().BoolVar(&stdinArg, "stdin", false, "Read data from stdin")
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "hash [flags] [files...]",
	Short:   "Hash and verify files",
	Version: version,
	// SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if stdinArg {
			var hash []byte

			hash, err = hashData(os.Stdin, algorithmArg)
			if err != nil {
				return
			}
			fmt.Println(hex.EncodeToString(hash))

			return
		}

		if len(args) < 1 {
			cmd.Help()
			return
		}

		for _, file := range args {
			var hash []byte

			hash, err = hashFile(file, algorithmArg)
			if err != nil {
				return
			}
			fmt.Printf("%s  %s\n", hex.EncodeToString(hash), file)
		}

		return
	},
}

func hashFile(filePath, algorithm string) (outputHash []byte, err error) {
	var file *os.File

	file, err = os.Open(filePath)
	if err != nil {
		err = fmt.Errorf("opening file %s: %w", filePath, err)
		return
	}
	defer file.Close()

	outputHash, err = hashData(file, algorithm)
	if err != nil {
		err = fmt.Errorf("file %s: %w", filePath, err)
		return
	}

	return
}

func hashData(data io.Reader, algorithm string) (outputHash []byte, err error) {
	var hasher hash.Hash

	switch algorithm {
	case AlgorithmSha256:
		hasher = sha256.New()
	case AlgorithmSha512:
		hasher = sha512.New()
	case AlgorithmBlake2b:
		hasher, err = blake2b.New(32, nil)
		if err != nil {
			err = fmt.Errorf("error when initializing blake2b hashing function: %w", err)
			return
		}
	case AlgorithmBlake3:
		hasher = blake3.New()

	default:
		err = fmt.Errorf("%s is not a supported hashing algorithm", algorithm)
		return
	}

	_, err = io.Copy(hasher, data)
	if err != nil {
		err = fmt.Errorf("error when hashing: %w", err)
		return
	}

	outputHash = hasher.Sum(nil)

	return
}
