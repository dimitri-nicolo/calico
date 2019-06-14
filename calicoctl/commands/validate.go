// Copyright (c) 2019 Tigera, Inc. All rights reserved.

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
	"io/ioutil"
	"os"
	"strings"

	"github.com/docopt/docopt-go"
	"github.com/tigera/licensing/client"
	"gopkg.in/yaml.v2"

	"github.com/projectcalico/calicoctl/calicoctl/commands/argutils"
	"github.com/projectcalico/calicoctl/calicoctl/commands/constants"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

func Validate(args []string) error {
	doc := constants.DatastoreIntro + `Usage:
  calicoctl validate --filename=<FILENAME>

Examples:
  # Validate the contents of license.yaml.
  calicoctl validate -f ./license.yaml

Options:
  -h --help                     Show this screen.
  -f --filename=<FILENAME>      Filename to validate.

Description:
  Validate a license file and report license status.

  The default output will be printed to stdout.
`
	parsedArgs, err := docopt.Parse(doc, args, true, "", false, false)
	if err != nil {
		return fmt.Errorf("Invalid option: 'calicoctl %s'. Use flag '--help' to read about a specific subcommand.", strings.Join(args, " "))
		os.Exit(1)
	}
	if len(parsedArgs) == 0 {
		return nil
	}

	filename := argutils.ArgStringOrBlank(parsedArgs, "--filename")

	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading license file '%v'", err)
	}

	lic := api.NewLicenseKey()

	err = yaml.Unmarshal(f, lic)
	if err != nil {
		return fmt.Errorf("error unmarshalling license file '%v'", err)
	}

	cl, err := client.Decode(*lic)
	if err != nil {
		return fmt.Errorf("error decoding license file '%v'", err)
	}

	licenseStatus := cl.Validate()
	fmt.Printf("License status: %s\n", licenseStatus.String())
	return nil
}
