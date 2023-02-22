// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.
//
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

package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

const ifaceModePath = "/etc/cni/net.d/calico_multi_interface_mode"
const ocIfaceModePath = "/run/multus/cni/net.d/calico_multi_interface_mode"

func SetEnv() error {
	filePath := ifaceModePath
	if _, err := os.Stat(ifaceModePath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		// this may be an openshift install, and if so this file will be somewhere else
		if _, err := os.Stat(ocIfaceModePath); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		filePath = ocIfaceModePath
	}

	modeBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	mode := strings.TrimSpace(string(modeBytes))

	if err := os.Setenv("MULTI_INTERFACE_MODE", mode); err != nil {
		return fmt.Errorf("error setting MULTI_INTERFACE_MODE	 environment variable: %v", err)
	}

	logrus.Debugf("set MULTI_INTERFACE_MODE to %s", mode)

	return nil
}
