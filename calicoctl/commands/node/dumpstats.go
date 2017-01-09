// Copyright (c) 2017 Tigera, Inc. All rights reserved.

package node

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/docopt/docopt-go"
	gops "github.com/mitchellh/go-ps"
)

func Dumpstats(args []string) {
	doc := `Usage:
  calicoctl node dumpstats

Options:
  -h --help                 Show this screen.

Description:
  Write the contents of calico-felix policy/rule counters to file pointed by
  the configuration parameter StatsDumpFilePath.
`

	parsedArgs, err := docopt.Parse(doc, args, true, "", false, false)
	if err != nil {
		fmt.Printf("Invalid option: 'calicoctl %s'. Use flag '--help' to read about a specific subcommand.\n", strings.Join(args, " "))
		os.Exit(1)
	}
	if len(parsedArgs) == 0 {
		return
	}

	// Root required for sending SIGUSR2 to felix.
	enforceRoot()

	processes, err := gops.Processes()
	if err != nil {
		fmt.Println(err)
	}
	proc := psGrep("calico-felix", processes)
	if proc == nil {
		fmt.Printf("calico-felix is not running.\n")
		os.Exit(1)
	}

	fmt.Printf("Sending SIGUSR2 to calico-felix.\n")
	syscall.Kill(proc.Pid(), syscall.SIGUSR2)
	fmt.Printf("Please check 'StatsDumpFilePath' for stats.\n")
}

func psGrep(proc string, procList []gops.Process) gops.Process {
	for _, p := range procList {
		if p.Executable() == proc {
			return p
		}
	}
	return nil
}
