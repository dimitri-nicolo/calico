package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(GenerateLicenseCmd)
}

var RootCmd = &cobra.Command{
	Use: "carrotctl",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("welcome to carrotctl :)")

	},
}