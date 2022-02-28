// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

package waf

import (
	"testing"
)

func TestCheckRulesSetExists_OK(t *testing.T) {

	err := CheckRulesSetExists(TestCoreRulesetDirectory)
	if err != nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
	}

	isEnabled := IsEnabled()
	if !isEnabled {
		t.Errorf("Expect: enabled 'true' Actual: '%v'", isEnabled)
	}
}

func TestCheckRulesSetExists_InvalidDirectory(t *testing.T) {

	err := CheckRulesSetExists(TestInvalidRulesetDirectory)
	if err == nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
	}

	isEnabled := IsEnabled()
	if isEnabled {
		t.Errorf("Expect: enabled 'false' Actual: '%v'", isEnabled)
	}
}

func TestCheckRulesSetExists_EmptyDirectory(t *testing.T) {

	err := CheckRulesSetExists(TestEmptyRulesetDirectory)
	if err == nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
	}

	isEnabled := IsEnabled()
	if isEnabled {
		t.Errorf("Expect: enabled 'false' Actual: '%v'", isEnabled)
	}
}

func TestDefineRulesSetDirectory(t *testing.T) {

	DefineRulesSetDirectory(TestCoreRulesetDirectory)
	expect := "../test/waf_test_files/core-rules/"
	actual := GetRulesDirectory()
	if expect != actual {
		t.Errorf("Expect: '%s' Actual: '%s'", expect, actual)
	}
}

func TestExtractRulesSetFilenamesCore(t *testing.T) {

	DefineRulesSetDirectory(TestCoreRulesetDirectory)

	expectFilenames := []string{
		"../test/waf_test_files/core-rules/modsecdefault.conf",
		"../test/waf_test_files/core-rules/crs-setup.conf",
		"../test/waf_test_files/core-rules/REQUEST-942-APPLICATION-ATTACK-SQLI.conf",
		"../test/waf_test_files/core-rules/REQUEST-901-INITIALIZATION.conf",
	}

	_ = ExtractRulesSetFilenames()
	actualFilenames := GetRulesSetFilenames()

	test := len(expectFilenames) == len(actualFilenames)
	if !test {
		t.Errorf("Expect '%s' Actual '%s'", expectFilenames, actualFilenames)
	}
}

func TestExtractRulesSetFilenamesCoreOrdered(t *testing.T) {

	DefineRulesSetDirectory(TestCoreRulesetDirectory)

	expectFilename := "../test/waf_test_files/core-rules/crs-setup.conf"

	_ = ExtractRulesSetFilenames()
	actualFilenames := GetRulesSetFilenames()
	actualFilename := actualFilenames[1]

	test := actualFilename == expectFilename
	if !test {
		t.Errorf("Expect '%s' Actual '%s'", expectFilename, actualFilenames)
	}
}

func TestExtractRulesSetFilenamesData(t *testing.T) {

	DefineRulesSetDirectory(TestDataRulesetDirectory)

	expectFilenames := []string{
		"../test/waf_test_files/data-rules/REQUEST-913-SCANNER-DETECTION.conf",
	}

	_ = ExtractRulesSetFilenames()
	actualFilenames := GetRulesSetFilenames()

	test := len(expectFilenames) == len(actualFilenames)
	if !test {
		t.Errorf("Expect '%s' Actual '%s'", expectFilenames, actualFilenames)
	}
}

func TestExtractRulesSetFilenamesInvalid(t *testing.T) {

	DefineRulesSetDirectory(TestInvalidRulesetDirectory)

	err := ExtractRulesSetFilenames()
	if err == nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
	}
}

func TestInitializeModSecurity(t *testing.T) {

	InitializeModSecurity()
}

func TestLoadModSecurityCoreRuleSetCore(t *testing.T) {

	InitializeModSecurity()
	filenames := []string{
		"../test/waf_test_files/core-rules/crs-setup.conf",
		"../test/waf_test_files/core-rules/modsecdefault.conf",
		"../test/waf_test_files/core-rules/REQUEST-942-APPLICATION-ATTACK-SQLI.conf",
	}

	err := LoadModSecurityCoreRuleSet(filenames)
	if err != nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
	}
}

func TestLoadModSecurityCoreRuleSetDataFiles(t *testing.T) {

	InitializeModSecurity()
	filenames := []string{
		"../test/waf_test_files/data-rules/REQUEST-913-SCANNER-DETECTION.conf",
	}

	err := LoadModSecurityCoreRuleSet(filenames)
	if err != nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
	}
}

func TestLoadModSecurityCoreRuleSetDataDirectory(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(TestDataRulesetDirectory)

	_ = ExtractRulesSetFilenames()
	filenames := GetRulesSetFilenames()

	err := LoadModSecurityCoreRuleSet(filenames)
	if err != nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
	}
}

func TestLoadModSecurityCoreRuleSetErrorDirectory(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(TestErrorRulesetDirectory)

	_ = ExtractRulesSetFilenames()
	filenames := GetRulesSetFilenames()

	err := LoadModSecurityCoreRuleSet(filenames)
	if err == nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
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
	DefineRulesSetDirectory(TestCoreRulesetDirectory)

	_ = ExtractRulesSetFilenames()
	filenames := GetRulesSetFilenames()
	_ = LoadModSecurityCoreRuleSet(filenames)

	id := "7ce62288-d6dd-4be0-8b31-ae27876aeeea"
	url := "/foo.com"
	httpMethod := "GET"
	httpProtocol := "HTTP"
	httpVersion := "1.1"
	clientHost := "http://localhost"
	clientPort := uint32(80)
	serverHost := "http://localhost"
	serverPort := uint32(80)

	err := ProcessHttpRequest(id, url, httpMethod, httpProtocol, httpVersion, clientHost, clientPort, serverHost, serverPort, nil, "")
	if err != nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
	}
}

func TestProcessHttpRequest_InvalidURL_BlockDueToWarning(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(TestCoreRulesetDirectory)

	_ = ExtractRulesSetFilenames()
	filenames := GetRulesSetFilenames()
	_ = LoadModSecurityCoreRuleSet(filenames)

	id := "7ce62288-d6dd-4be0-8b31-ae27876aeeea"
	url := "/test/artists.php?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user"
	httpMethod := "GET"
	httpProtocol := "HTTP"
	httpVersion := "1.1"
	clientHost := "http://localhost"
	clientPort := uint32(80)
	serverHost := "http://localhost"
	serverPort := uint32(80)

	err := ProcessHttpRequest(id, url, httpMethod, httpProtocol, httpVersion, clientHost, clientPort, serverHost, serverPort, nil, "")
	if err != nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
	}
}

func TestProcessHttpRequest_InvalidURL_NoRulesLoad_OK(t *testing.T) {

	InitializeModSecurity()
	var filenames []string
	_ = LoadModSecurityCoreRuleSet(filenames)

	id := "7ce62288-d6dd-4be0-8b31-ae27876aeeea"
	url := "/test/artists.php?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user"
	httpMethod := "GET"
	httpProtocol := "HTTP"
	httpVersion := "1.1"
	clientHost := "http://localhost"
	clientPort := uint32(80)
	serverHost := "http://localhost"
	serverPort := uint32(80)

	err := ProcessHttpRequest(id, url, httpMethod, httpProtocol, httpVersion, clientHost, clientPort, serverHost, serverPort, nil, "")
	if err != nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
	}
}

func TestProcessHttpRequest_InvalidURL_CustomRulesLoad_BadRequest(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(TestCustomRulesetDirectory)

	_ = ExtractRulesSetFilenames()
	filenames := GetRulesSetFilenames()
	_ = LoadModSecurityCoreRuleSet(filenames)

	id := "7ce62288-d6dd-4be0-8b31-ae27876aeeea"
	url := "/test/artists.php?artist=0+div+1+union%23foo*%2F*bar%0D%0Aselect%23foo%0D%0A1%2C2%2Ccurrent_user"
	httpMethod := "GET"
	httpProtocol := "HTTP"
	httpVersion := "1.1"
	clientHost := "http://localhost"
	clientPort := uint32(80)
	serverHost := "http://localhost"
	serverPort := uint32(80)

	err := ProcessHttpRequest(id, url, httpMethod, httpProtocol, httpVersion, clientHost, clientPort, serverHost, serverPort, nil, "")
	if err == nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
	}
}

func TestProcessHttpRequest_AddRequestInfo_CoreRulesDenyLoad_OK(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(TestCoreRulesetDenyDirectory)

	_ = ExtractRulesSetFilenames()
	filenames := GetRulesSetFilenames()
	_ = LoadModSecurityCoreRuleSet(filenames)

	id := "7ce62288-d6dd-4be0-8b31-ae27876aeeea"
	url := "/"
	httpMethod := "POST"
	httpProtocol := "HTTP"
	httpVersion := "1.1"
	clientHost := "http://localhost"
	clientPort := uint32(80)
	serverHost := "http://localhost"
	serverPort := uint32(80)
	reqHeaders := map[string]string{
		"content-type": "application/xml",
		"User-Agent":   "Microsoft Internet Explorer",
	}
	reqBody := "{\"productId\": 123456, \"quantity\": 100}"

	err := ProcessHttpRequest(id, url, httpMethod, httpProtocol, httpVersion, clientHost, clientPort, serverHost, serverPort, reqHeaders, reqBody)
	if err != nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
	}
}

func TestProcessHttpRequest_AddRequestInfo_CoreRulesDenyLoad_BadRequest(t *testing.T) {

	InitializeModSecurity()
	DefineRulesSetDirectory(TestCustomRulesetDirectory)

	_ = ExtractRulesSetFilenames()
	filenames := GetRulesSetFilenames()
	_ = LoadModSecurityCoreRuleSet(filenames)

	id := "7ce62288-d6dd-4be0-8b31-ae27876aeeea"
	url := "/"
	httpMethod := "POST"
	httpProtocol := "HTTP"
	httpVersion := "1.1"
	clientHost := "http://localhost"
	clientPort := uint32(80)
	serverHost := "http://localhost"
	serverPort := uint32(80)
	reqHeaders := map[string]string{
		"content-type": "application/x-www-form-urlencoded",
	}
	reqBody := "<script>alert(1)</script>"

	err := ProcessHttpRequest(id, url, httpMethod, httpProtocol, httpVersion, clientHost, clientPort, serverHost, serverPort, reqHeaders, reqBody)
	if err == nil {
		t.Errorf("Expect: error 'nil' Actual: '%v'", err.Error())
	}
}

func TestCleanupModSecurity(t *testing.T) {

	CleanupModSecurity()
}
