// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

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

package waf

import (
	"testing"
)

const testCoreRulesetDirectory = "test_files/core-rules"
const testCustomRulesetDirectory = "test_files/custom-rules"
const testDataRulesetDirectory = "test_files/data-rules"

func TestInitializeModSecurity(t *testing.T) {

	InitializeModSecurity()
}

func TestDefineRulesSetDirectory(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(testCoreRulesetDirectory)
	expect := "test_files/core-rules/"
	actual := GetRulesDirectory()
	if expect != actual {
		t.Errorf("Expect: '%s' Actual: '%s'", expect, actual)
	}
}

func TestExtractRulesSetFilenamesCore(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(testCoreRulesetDirectory)

	expectFilenames := []string{
		"test_files/core-rules/modsecdefault.conf",
		"test_files/core-rules/crs-setup.conf",
		"test_files/core-rules/REQUEST-942-APPLICATION-ATTACK-SQLI.conf",
		"test_files/core-rules/REQUEST-901-INITIALIZATION.conf",
	}
	actualFilenames := ExtractRulesSetFilenames()

	test := len(expectFilenames) == len(actualFilenames)
	if !test {
		t.Errorf("Expect '%s' Actual '%s'", expectFilenames, actualFilenames)
	}
}

func TestExtractRulesSetFilenamesCoreOrdered(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(testCoreRulesetDirectory)

	expectFilename := "test_files/core-rules/crs-setup.conf"

	actualFilenames := ExtractRulesSetFilenames()
	actualFilename := actualFilenames[1]

	test := actualFilename == expectFilename
	if !test {
		t.Errorf("Expect '%s' Actual '%s'", expectFilename, actualFilenames)
	}
}

func TestExtractRulesSetFilenamesData(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(testDataRulesetDirectory)

	expectFilenames := []string{
		"test_files/data-rules/REQUEST-913-SCANNER-DETECTION.conf",
	}
	actualFilenames := ExtractRulesSetFilenames()

	test := len(expectFilenames) == len(actualFilenames)
	if !test {
		t.Errorf("Expect '%s' Actual '%s'", expectFilenames, actualFilenames)
	}
}

func TestLoadModSecurityCoreRuleSetCore(t *testing.T) {

	InitializeModSecurity()
	filenames := []string{
		"test_files/core-rules/crs-setup.conf",
		"test_files/core-rules/modsecdefault.conf",
		"test_files/core-rules/REQUEST-942-APPLICATION-ATTACK-SQLI.conf",
	}

	expect := len(filenames)
	actual := LoadModSecurityCoreRuleSet(filenames)

	if expect != actual {
		t.Errorf("Expect: %d Actual: %d", expect, actual)
	}
}

func TestLoadModSecurityCoreRuleSetDataFiles(t *testing.T) {

	InitializeModSecurity()
	filenames := []string{
		"test_files/data-rules/REQUEST-913-SCANNER-DETECTION.conf",
	}

	expect := len(filenames)
	actual := LoadModSecurityCoreRuleSet(filenames)

	if expect != actual {
		t.Errorf("Expect: %d Actual: %d", expect, actual)
	}
}

func TestLoadModSecurityCoreRuleSetDataDirectory(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(testDataRulesetDirectory)

	expectFilenames := []string{
		"test_files/data-rules/REQUEST-913-SCANNER-DETECTION.conf",
	}
	expect := len(expectFilenames)

	actualFilenames := ExtractRulesSetFilenames()
	actual := LoadModSecurityCoreRuleSet(actualFilenames)

	if expect != actual {
		t.Errorf("Expect: %d Actual: %d", expect, actual)
	}
}

func TestLoadModSecurityCoreRuleSetErrorDirectory(t *testing.T) {

	InitializeModSecurity()
	filenames := []string{
		"test_files/error-rules/REQUEST-941-APPLICATION-ATTACK-XSS.conf",
		"test_files/error-rules/REQUEST-942-APPLICATION-ATTACK-SQLI.conf",
	}

	expect := 0
	actual := LoadModSecurityCoreRuleSet(filenames)

	if expect != actual {
		t.Errorf("Expect: %d Actual: %d", expect, actual)
	}
}

func TestGenerateModSecurityID(t *testing.T) {

	id := GenerateModSecurityID()
	expectLength := 36
	actualLength := len(id)

	if expectLength != actualLength {
		t.Errorf("ID '%s' expect length: %d actual length: %d", id, expectLength, actualLength)
	}
}

func TestProcessHttpRequest_ValidURL_OK(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(testCoreRulesetDirectory)
	filenames := ExtractRulesSetFilenames()
	LoadModSecurityCoreRuleSet(filenames)

	id := "7ce62288-d6dd-4be0-8b31-ae27876aeeea"
	url := "/foo.com"
	httpMethod := "GET"
	httpProtocol := "HTTP"
	httpVersion := "1.1"
	clientHost := "http://localhost"
	clientPort := uint32(80)
	serverHost := "http://localhost"
	serverPort := uint32(80)

	expect := 0
	actual := ProcessHttpRequest(id, url, httpMethod, httpProtocol, httpVersion, clientHost, clientPort, serverHost, serverPort)

	if expect != actual {
		t.Errorf("Expect: %d Actual: %d", expect, actual)
	}
}

func TestProcessHttpRequest_InvalidURL_BlockDueToWarning(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(testCoreRulesetDirectory)
	filenames := ExtractRulesSetFilenames()
	LoadModSecurityCoreRuleSet(filenames)

	id := "7ce62288-d6dd-4be0-8b31-ae27876aeeea"
	url := "/test/artists.php?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user"
	httpMethod := "GET"
	httpProtocol := "HTTP"
	httpVersion := "1.1"
	clientHost := "http://localhost"
	clientPort := uint32(80)
	serverHost := "http://localhost"
	serverPort := uint32(80)

	expect := 0
	actual := ProcessHttpRequest(id, url, httpMethod, httpProtocol, httpVersion, clientHost, clientPort, serverHost, serverPort)

	if expect != actual {
		t.Errorf("Expect: %d Actual: %d", expect, actual)
	}
}

func TestProcessHttpRequest_InvalidURL_NoRulesLoad_OK(t *testing.T) {

	InitializeModSecurity()
	var filenames []string
	LoadModSecurityCoreRuleSet(filenames)

	id := "7ce62288-d6dd-4be0-8b31-ae27876aeeea"
	url := "/test/artists.php?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user"
	httpMethod := "GET"
	httpProtocol := "HTTP"
	httpVersion := "1.1"
	clientHost := "http://localhost"
	clientPort := uint32(80)
	serverHost := "http://localhost"
	serverPort := uint32(80)

	expect := 0
	actual := ProcessHttpRequest(id, url, httpMethod, httpProtocol, httpVersion, clientHost, clientPort, serverHost, serverPort)

	if expect != actual {
		t.Errorf("Expect: %d Actual: %d", expect, actual)
	}
}

func TestProcessHttpRequest_InvalidURL_CustomRulesLoad_BadRequest(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(testCustomRulesetDirectory)
	filenames := ExtractRulesSetFilenames()
	LoadModSecurityCoreRuleSet(filenames)

	id := "7ce62288-d6dd-4be0-8b31-ae27876aeeea"
	url := "/test/artists.php?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user"
	httpMethod := "GET"
	httpProtocol := "HTTP"
	httpVersion := "1.1"
	clientHost := "http://localhost"
	clientPort := uint32(80)
	serverHost := "http://localhost"
	serverPort := uint32(80)

	expect := 1
	actual := ProcessHttpRequest(id, url, httpMethod, httpProtocol, httpVersion, clientHost, clientPort, serverHost, serverPort)

	if expect != actual {
		t.Errorf("Expect: %d Actual: %d", expect, actual)
	}
}

func TestCleanupModSecurity(t *testing.T) {

	CleanupModSecurity()
}
