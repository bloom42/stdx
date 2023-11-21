package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/bloom42/stdx/cobra"
	"github.com/bloom42/stdx/crypto"
)

const (
	defaultPasswordLength = 128
	version               = "1.0.0"
)

const (
	// LowerLetters is the list of lowercase letters.
	LowerLetters = "abcdefghijklmnopqrstuvwxyz"

	// UpperLetters is the list of uppercase letters.
	UpperLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	// Digits is the list of permitted digits.
	Digits = "0123456789"

	// Symbols is the list of symbols.
	Symbols = "~!@#$%^&*()_+`-={}|[]\\:\"<>?,./"

	All = LowerLetters + UpperLetters + Digits + Symbols
)

var (
	passwordLength uint64
)

func init() {
	rootCmd.Flags().Uint64VarP(&passwordLength, "length", "n", defaultPasswordLength, "Password length (default: 128, min: 8, max: 4096)")
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:           "password",
	Short:         "Generate secure passwords",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if passwordLength < 8 {
			err = errors.New("Password cannot be shorter than 8 characters")
			return
		} else if passwordLength > 4096 {
			err = errors.New("Password cannot be longer than 4096 characters")
			return
		}

		password, err := crypto.RandAlphabet([]byte(All), passwordLength)
		if err != nil {
			return
		}
		fmt.Println(string(password))

		return
	},
}
