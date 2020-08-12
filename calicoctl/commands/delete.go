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
	"fmt"
	"strings"

	"github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calicoctl/calicoctl/commands/common"
	"github.com/projectcalico/calicoctl/calicoctl/commands/constants"
)

func Delete(args []string) error {
	doc := constants.DatastoreIntro + `Usage:
  calicoctl delete ( (<KIND> [<NAME>...]) |
                   --filename=<FILE> [--recursive])
                   [--skip-not-exists] [--config=<CONFIG>] [--namespace=<NS>]

Examples:
  # Delete a policy using the type and name specified in policy.yaml.
  calicoctl delete -f ./policy.yaml

  # Delete a policy based on the type and name in the YAML passed into stdin.
  cat policy.yaml | calicoctl delete -f -

  # Delete policies with names "foo" and "bar"
  calicoctl delete policy foo bar

Options:
  -h --help                 Show this screen.
  -s --skip-not-exists      Skip over and treat as successful, resources that
                            don't exist.
  -f --filename=<FILENAME>  Filename to use to delete the resource.  If set to
                            "-" loads from stdin. If filename is a directory, this command is
                            invoked for each .json .yaml and .yml file within that directory,
                            terminating after the first failure.
  -R --recursive            Process the filename specified in -f or --filename recursively.
  -c --config=<CONFIG>      Path to the file containing connection
                            configuration in YAML or JSON format.
                            [default: ` + constants.DefaultConfigPath + `]
  -n --namespace=<NS>       Namespace of the resource.
                            Only applicable to NetworkPolicy, StagedNetworkPolicy, StagedKubernetesNetworkPolicy,
                            NetworkSet, and WorkloadEndpoint.
                            Uses the default namespace if not specified.

Description:
  The delete command is used to delete a set of resources by filename or stdin,
  or by type and identifiers.  JSON and YAML formats are accepted for file and
  stdin format.
  
  Valid resource types are:

    * bgpConfiguration
    * bgpPeer
    * felixConfiguration
    * globalNetworkPolicy
    * stagedGlobalNetworkPolicy
    * globalNetworkSet
    * globalThreatFeed
    * hostEndpoint
    * ipPool
    * kubeControllersConfiguration
    * tier
    * networkPolicy
    * stagedNetworkPolicy
    * stagedKubernetesNetworkPolicy
    * networkSet
    * node
    * profile
    * workloadEndpoint
    * remoteClusterConfiguration

  The resource type is case insensitive and may be pluralized.

  Attempting to delete a resource that does not exists is treated as a
  terminating error unless the --skip-not-exists flag is set.  If this flag is
  set, resources that do not exist are skipped.

  When deleting resources by type, only a single type may be specified at a
  time.  The name is required along with any and other identifiers required to
  uniquely identify a resource of the specified type.

  The output of the command indicates how many resources were successfully
  deleted, and the error reason if an error occurred.  If the --skip-not-exists
  flag is set then skipped resources are included in the success count.

  The resources are deleted in the order they are specified.  In the event of a
  failure deleting a specific resource it is possible to work out which
  resource failed based on the number of resources successfully deleted.
`
	parsedArgs, err := docopt.Parse(doc, args, true, "", false, false)
	if err != nil {
		return fmt.Errorf("Invalid option: 'calicoctl %s'. Use flag '--help' to read about a specific subcommand.", strings.Join(args, " "))
	}
	if len(parsedArgs) == 0 {
		return nil
	}

	results := common.ExecuteConfigCommand(parsedArgs, common.ActionDelete)
	log.Infof("results: %+v", results)

	if results.FileInvalid {
		return fmt.Errorf("Failed to execute command: %v", results.Err)
	} else if results.NumResources == 0 {
		return fmt.Errorf("No resources specified")
	} else if results.Err == nil && results.NumHandled > 0 {
		if results.SingleKind != "" {
			fmt.Printf("Successfully deleted %d '%s' resource(s)\n", results.NumHandled, results.SingleKind)
		} else {
			fmt.Printf("Successfully deleted %d resource(s)\n", results.NumHandled)
		}
	} else if results.Err != nil {
		return fmt.Errorf("Hit error: %v", results.Err)
	}

	if len(results.ResErrs) > 0 {
		var errStr string
		for _, err := range results.ResErrs {
			if results.SingleKind != "" {
				errStr += fmt.Sprintf("Failed to delete '%s' resource: %v\n", results.SingleKind, err)
			} else {
				errStr += fmt.Sprintf("Failed to delete resource: %v\n", err)
			}
		}
		return fmt.Errorf(errStr)
	}

	return nil
}
