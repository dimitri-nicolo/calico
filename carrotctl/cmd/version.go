package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var VERSION, BUILD_DATE, GIT_REVISION string

func init()  {

}

var VersionCmd = &cobra.Command{
	Use:        "version",
	Aliases:    []string{"version", "ver", "who-dis"},
	SuggestFor: []string{"versio", "covfefe"},
	Short:      "carrotctl version",
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("Build Version:    ", VERSION)
		fmt.Println("Build date:       ", BUILD_DATE)
		fmt.Println("Git commit:       ", GIT_REVISION)
	},
}
