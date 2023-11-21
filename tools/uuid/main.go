package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/bloom42/stdx/cobra"
	"github.com/bloom42/stdx/uuid"
)

const (
	defautlNumberOfUuids = 1
	version              = "1.0.0"
)

var (
	numberOfUuids uint64
)

func init() {
	rootCmd.Flags().Uint64VarP(&numberOfUuids, "number", "n", defautlNumberOfUuids, "Number of UUIDs to generate (default: 1)")
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stdout, err.Error())
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:           "uuid",
	Short:         "Generate UUIDs",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if numberOfUuids > 2_147_483_647 {
			err = errors.New("Can't generate more than 2,147,483,647 UUIDs")
			return
		}

		for i := uint64(0); i < numberOfUuids; i += 1 {
			uuid := uuid.New()
			fmt.Println(uuid.String())
		}

		return
	},
}
