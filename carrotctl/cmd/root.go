package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(GenerateLicenseCmd)
	RootCmd.AddCommand(ListLicensesCmd)
	RootCmd.AddCommand(RetrieveLicenseCmd)
	RootCmd.AddCommand(VersionCmd)
}

var RootCmd = &cobra.Command{
	Use: "carrotctl",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("carrotctl <command> [<args>...]")
		fmt.Println("enter 'carrotctl -h' for help")
	},
}
