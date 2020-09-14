// Copyright (c) 2016-2020 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docopt/docopt-go"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/libcalico-go/lib/options"

	"github.com/projectcalico/calicoctl/calicoctl/commands/argutils"
	"github.com/projectcalico/calicoctl/calicoctl/commands/clientmgr"
	"github.com/projectcalico/calicoctl/calicoctl/commands/constants"
)

var VERSION, GIT_REVISION string
var VERSION_SUMMARY string

func init() {
	VERSION_SUMMARY = `Run 'calicoctl version' to see version information.`
}

func Version(args []string) error {
	doc := `Usage:
  calicoctl version [--config=<CONFIG>] [--poll=<POLL>]

Options:
  -h --help             Show this screen.
  -c --config=<CONFIG>  Path to the file containing connection configuration in
                        YAML or JSON format.
                        [default: ` + constants.DefaultConfigPath + `]
     --poll=<POLL>      Poll for changes to the cluster information at a frequency specified using POLL duration
                        (e.g. 1s, 10m, 2h etc.). A value of 0 (the default) disables polling.

Description:
  Display the version of calicoctl.
`
	parsedArgs, err := docopt.Parse(doc, args, true, "", false, false)
	if err != nil {
		return fmt.Errorf("Invalid option: 'calicoctl %s'. Use flag '--help' to read about a specific subcommand.", strings.Join(args, " "))
	}
	if len(parsedArgs) == 0 {
		return nil
	}

	// Parse the poll duration.
	var pollDuration time.Duration
	var ci *v3.ClusterInformation
	if poll := argutils.ArgStringOrBlank(parsedArgs, "--poll"); poll != "" {
		if pollDuration, err = time.ParseDuration(poll); err != nil {
			return fmt.Errorf("Invalid poll duration specified: %s", pollDuration)
		}
	}

	fmt.Println("Client Version:   ", VERSION)
	fmt.Println("Release:          ", "Calico Enterprise")
	fmt.Println("Git commit:       ", GIT_REVISION)

	// Load the client config and connect.
	cf := parsedArgs["--config"].(string)
	client, err := clientmgr.NewClient(cf)
	if err != nil {
		return err
	}
	ctx := context.Background()
	var pv, pt string
	var pcv string

	for {
		if ci, err = client.ClusterInformation().Get(ctx, "default", options.GetOptions{}); err == nil {
			v := ci.Spec.CalicoVersion
			if v == "" {
				v = "unknown"
			}
			cv := ci.Spec.CNXVersion
			if cv == "" {
				cv = "unknown"
			}
			t := ci.Spec.ClusterType
			if t == "" {
				t = "unknown"
			}

			if pv != v {
				fmt.Println("Cluster Calico Version:              ", v)
				pv = v
			}
			if pcv != cv {
				fmt.Println("Cluster Calico Enterprise Version:   ", cv)
				pcv = cv
			}
			if pt != t {
				fmt.Println("Cluster Type:                        ", t)
				pt = t
			}
		} else {
			// Unable to retrieve the version.  Reset the old versions so that we re-display when we are able to
			// determine the version again (if polling).
			err = fmt.Errorf("Unable to retrieve Cluster Version or Type: %s", err)
			pv = ""
			pt = ""
		}

		if pollDuration == 0 {
			// We are not polling, so exit.
			break
		}

		// We are polling, so display any error that we encountered determining the version and then wait for the next
		// iteration.
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		time.Sleep(pollDuration)
	}

	return err
}
