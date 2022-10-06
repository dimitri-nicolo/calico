// Copyright (c) 2022 Tigera, Inc. All rights reserved.
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

package server

import (
	"os"
	"testing"
)

func TestCATypeFlagParsing(t *testing.T) {
	testCases := []struct {
		args                []string
		expectedParsedValue string
	}{
		// When the CA type flag is not set, the parsed value should be "" to ensure the tls section is not rendered
		// in the generated ManagementClusterConnection manifest (EV-2618)
		{[]string{}, ""},
		{[]string{"--managementClusterCAType=Tigera"}, "Tigera"},
		{[]string{"--managementClusterCAType=Public"}, "Public"},
	}

	for _, testCase := range testCases {
		cmd, err := NewCommandStartCalicoServer(os.Stdout)
		if err != nil {
			t.Fatalf("Failed to create the server command: %v", err)
		}

		err = cmd.ParseFlags(testCase.args)
		if err != nil {
			t.Fatalf("Failed to parse flags from the server command: %v", err)
		}

		parsedValue, err := cmd.Flags().GetString("managementClusterCAType")
		if err != nil {
			t.Fatalf("Failed to get managementClusterCAType flag from the server command: %v", err)
		}

		if parsedValue != testCase.expectedParsedValue {
			t.Fatalf(
				"Parsed value %v for managementClusterCAType flag, expected %v for args %v",
				parsedValue,
				testCase.expectedParsedValue,
				testCase.args,
			)
		}
	}
}
