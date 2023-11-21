package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bloom42/stdx/cobra"
	"github.com/bloom42/stdx/crypto"
	"github.com/bloom42/stdx/filesutil"
	"github.com/bloom42/stdx/zign"
	"golang.org/x/term"
)

const (
	defaultPrivateKeyFile = "zign.private"
	defaultPulicKeyFile   = "zign.public"
)

var (
	signManifestOutpout  string
	signCmdPasswordStdin bool

	version = fmt.Sprintf("%d.0.0", zign.Version1)
)

func init() {
	rootCmd.AddCommand(initCmd)

	signCmd.Flags().StringVarP(&signManifestOutpout, "output", "o", zign.DefaultManifestFilename, "Output file for signature (default: zign.json)")
	signCmd.Flags().BoolVar(&signCmdPasswordStdin, "password-stdin", false, "Read password from stdin")
	rootCmd.AddCommand(signCmd)

	rootCmd.AddCommand(verifyCmd)
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:           "zign",
	Short:         "Sign and verify files",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		return cmd.Help()
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a keypair to sign files (zign.private, zign.public)",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		fmt.Print("password: ")
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			err = fmt.Errorf("zign: error reading password: %w", err)
			return
		}
		defer crypto.Zeroize(password)

		fmt.Print("\nverify password: ")
		passwordVerification, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			err = fmt.Errorf("zign: error reading password: %w", err)
			return
		}
		defer crypto.Zeroize(passwordVerification)

		if crypto.ConstantTimeCompare(password, passwordVerification) == false {
			err = errors.New("zign: passwords don't match")
			return
		}

		fmt.Println("")

		privateKey, publicKey, err := zign.Init(password)
		if err != nil {
			return
		}

		privateKey += "\n"

		err = os.WriteFile(defaultPrivateKeyFile, []byte(privateKey), 0600)
		if err != nil {
			err = fmt.Errorf("zign: writing private key file: %w", err)
			return
		}

		publicKey += "\n"

		err = os.WriteFile(defaultPulicKeyFile, []byte(publicKey), 0600)
		if err != nil {
			err = fmt.Errorf("zign: writing public key file: %w", err)
			return
		}

		return
	},
}

var signCmd = &cobra.Command{
	Use:   "sign [-o project_version_zign.json] zign.private file1 file2 file3...",
	Short: "Sign files using the given private key",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var password []byte

		if len(args) < 2 {
			return cmd.Help()
		}

		if signCmdPasswordStdin {
			stdinReader := bufio.NewReader(os.Stdin)
			password, _, err = stdinReader.ReadLine()
			if err != nil {
				err = fmt.Errorf("zign: error reading password from stdin: %w", err)
				return
			}
		} else {
			fmt.Print("password: ")
			password, err = term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				err = fmt.Errorf("zign: error reading password: %w", err)
				return
			}

			fmt.Println("")
		}
		defer crypto.Zeroize(password)

		encodedPrivateKeyBytes, err := os.ReadFile(defaultPrivateKeyFile)
		if err != nil {
			err = fmt.Errorf("zign: reading private key file: %w", err)
			return
		}

		encodedPrivateKey := strings.TrimSpace(string(encodedPrivateKeyBytes))

		files := args[1:]
		signInput := make([]zign.SignInput, len(files))

		for index, file := range files {
			fileExists := false
			var fileHandle *os.File

			fileExists, err = filesutil.Exists(file)
			if err != nil {
				err = fmt.Errorf("checking if file exists (%s): %w", file, err)
				return
			}

			if !fileExists {
				err = fmt.Errorf("file does not exist: %s", file)
				return
			}

			// we don't need to close the files as the program is short lived...
			fileHandle, err = os.Open(file)
			if err != nil {
				err = fmt.Errorf("opening file (%s): %w", file, err)
				return
			}
			fileSignInput := zign.SignInput{
				Reader: fileHandle,
			}
			signInput[index] = fileSignInput
		}

		signOutput, err := zign.SignMany(encodedPrivateKey, string(password), signInput)
		if err != nil {
			return
		}

		manifest := zign.GenerateManifest(signOutput)
		manifestJson, err := manifest.ToJson()
		if err != nil {
			return
		}

		err = os.WriteFile(signManifestOutpout, manifestJson, 0644)
		if err != nil {
			err = fmt.Errorf("zign: writing manifest (%s): %w", signManifestOutpout, err)
			return
		}

		return
	},
}

var verifyCmd = &cobra.Command{
	Use:   "verify [base64_encoded_public_key] project_version_zign.json",
	Short: "Verify files using the given public key",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var manifest zign.Manifest
		if len(args) != 2 {
			return cmd.Help()
		}

		manifestJson, err := os.ReadFile(args[1])
		if err != nil {
			return
		}

		err = json.Unmarshal(manifestJson, &manifest)
		if err != nil {
			err = fmt.Errorf("parsing manifest: %w", err)
			return
		}

		if manifest.Version != zign.Version1 {
			err = fmt.Errorf("zign: manifest version superior to %s are not supported", zign.Version1)
			return
		}

		verifyInput := make([]zign.VerifyInput, 0, len(manifest.Files))
		verifiedFiles := make([]string, 0, len(manifest.Files))

		for _, file := range manifest.Files {
			fileExists := false
			var fileHandle *os.File
			var sha256Hash []byte

			fileExists, err = filesutil.Exists(file.Filename)
			if err != nil {
				err = fmt.Errorf("checking if file exists (%s): %w", file.Filename, err)
				return
			}

			if !fileExists {
				continue
			}

			// we don't need to close the files as the program is short lived...
			fileHandle, err = os.Open(file.Filename)
			if err != nil {
				err = fmt.Errorf("opening file (%s): %w", file.Filename, err)
				return
			}

			sha256Hash, err = hex.DecodeString(file.Sha256)
			if err != nil {
				err = fmt.Errorf("decoding sha256 hash for file %s: %w", file.Filename, err)
				return
			}

			fileVerifyInput := zign.VerifyInput{
				Reader:    fileHandle,
				Sha256:    sha256Hash,
				Signature: file.Signature,
			}
			verifyInput = append(verifyInput, fileVerifyInput)
			verifiedFiles = append(verifiedFiles, file.Filename)
		}

		base64EncodedPublicKey := strings.TrimSpace(args[0])
		err = zign.VerifyMany(base64EncodedPublicKey, verifyInput)
		if err != nil {
			return
		}

		if len(verifiedFiles) == 0 {
			fmt.Println("No file to verify")
			return
		}

		for _, file := range verifiedFiles {
			fmt.Println("✓", file)
		}

		return
	},
}
