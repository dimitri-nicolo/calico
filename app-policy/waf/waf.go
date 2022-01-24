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

// #cgo CFLAGS: -I/usr/local/modsecurity/include
// #cgo LDFLAGS: -L/usr/local/modsecurity/lib/ -Wl,-rpath -Wl,/usr/local/modsecurity/lib/ -lmodsecurity
// #include "waf.h"
import "C"
import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"
	"unsafe"

	"github.com/google/uuid"

	log "github.com/sirupsen/logrus"
)

// Directory where the Core Rules Set are stored.
var rulesetDirectory string

const defaultRulesetDirectory = "/etc/waf/"

func InitializeModSecurity() {
	log.Printf("WAF Initialize Mod Security.")
	C.InitializeModSecurity()
}

func DefineRulesSetDirectory(directory string) {
	rulesetDirectory = directory

	// Defend against root access "/".
	if len(rulesetDirectory) < 2 {
		rulesetDirectory = defaultRulesetDirectory
	}
	log.Printf("WAF Core Rules Set directory: '%s'", rulesetDirectory)

	// Ensure rules directory ends with trailing slash.
	if !strings.HasSuffix(rulesetDirectory, "/") {
		rulesetDirectory = rulesetDirectory + "/"
	}
}

func ExtractRulesSetFilenames() []string {

	// Read all core rule set file names from rules directory.
	var files []string
	items, _ := ioutil.ReadDir(rulesetDirectory)

	// Sort files descending to ensure lower cased files like crs-setup.conf are loaded first.
	// This is a requirement for Core Rules Set and REQUEST-901-INITIALIZATION.conf bootstrap.
	sortFileNameDescend(items)

	count := 1
	for _, item := range items {

		// Ignore files starting with ".." that link to the parent directory.
		filename := item.Name()
		if strings.HasPrefix(filename, "..") {
			continue
		}

		// Only load *.conf configuration files.
		if !strings.HasSuffix(filename, ".conf") {
			continue
		}

		file := rulesetDirectory + filename
		files = append(files, file)
		log.Infof("WAF Found Rule[%d]('%s')", count, file)
		count++
	}

	log.Infof("WAF Total Core Rules Sets: %d", len(files))
	return files
}

func LoadModSecurityCoreRuleSet(filenames []string) int {

	size := len(filenames)
	log.Infof("WAF Attempt load %d Core Rule Sets", size)

	index := loadModSecurityCoreRuleSetImpl(filenames, size)
	if index == size {
		log.Infof("WAF Process load %d Core Rule Sets  SUCCESS", size)
	} else {
		badFile := filenames[index]
		log.Errorf("WAF Process load %d Core Rule Sets  FAILED!  Bad File: '%s'", size, badFile)
	}

	return index
}
func loadModSecurityCoreRuleSetImpl(filenames []string, size int) int {

	// Transfer core rule set file names to WAF wrapper code.
	csize := C.int(size)
	carray := C.makeCharArray(csize)
	defer C.freeCharArray(carray, csize)
	for index, filename := range filenames {
		C.setArrayString(carray, C.CString(filename), C.int(index))
	}

	// Finally, load ModSecurity core rule set from WAF wrapper code.
	return int(C.LoadModSecurityCoreRuleSet(carray, csize))
}

func GenerateModSecurityID() string {
	return uuid.New().String()
}

func ProcessHttpRequest(id, url, httpMethod, httpProtocol, httpVersion string, clientHost string, clientPort uint32, serverHost string, serverPort uint32) int {
	prefix := getProcessHttpRequestPrefix(id)
	log.Printf("%s URL '%s'", prefix, url)

	Cid := C.CString(id)
	Curi := C.CString(url)
	ChttpMethod := C.CString(httpMethod)
	ChttpProtocol := C.CString(httpProtocol)
	ChttpVersion := C.CString(httpVersion)
	CclientHost := C.CString(clientHost)
	CclientPort := C.int(clientPort)
	CserverHost := C.CString(serverHost)
	CserverPort := C.int(serverPort)

	defer C.free(unsafe.Pointer(Cid))
	defer C.free(unsafe.Pointer(Curi))
	defer C.free(unsafe.Pointer(ChttpMethod))
	defer C.free(unsafe.Pointer(ChttpProtocol))
	defer C.free(unsafe.Pointer(ChttpVersion))
	defer C.free(unsafe.Pointer(CclientHost))
	defer C.free(unsafe.Pointer(CserverHost))

	start := time.Now()
	detection := int(C.ProcessHttpRequest(Cid, Curi, ChttpMethod, ChttpProtocol, ChttpVersion, CclientHost, CclientPort, CserverHost, CserverPort))
	elapsed := time.Since(start)

	log.Infof("%s URL '%s' Detection=%d Time elapsed: %s", prefix, url, detection, elapsed)
	return detection
}

// GetRulesDirectory public helper function for testing.
func GetRulesDirectory() string {
	return rulesetDirectory
}

func getProcessHttpRequestPrefix(id string) string {
	return fmt.Sprintf("WAF Process Http Request [%s]", id)
}

func sortFileNameDescend(files []os.FileInfo) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() > files[j].Name()
	})
}

//export GoModSecurityLoggingCallback
func GoModSecurityLoggingCallback(Cpayload *C.char) {

	payload := C.GoString(Cpayload)
	dictionary := ParseLog(payload)

	// Log to Elasticsearch => Kibana.
	Logger.WithFields(log.Fields{
		"unique_id":      dictionary[ParserUniqueId],
		"uri":            dictionary[ParserUri],
		"owasp_host":     dictionary[ParserHostname],
		"owasp_file":     dictionary[ParserFile],
		"owasp_line":     dictionary[ParserLine],
		"owasp_id":       dictionary[ParserId],
		"owasp_data":     dictionary[ParserData],
		"owasp_severity": dictionary[ParserSeverity],
		"owasp_version":  dictionary[ParserVersion],
	}).Warn("WAF " + dictionary[ParserMsg])
}

func CleanupModSecurity() {
	C.CleanupModSecurity()
	log.Printf("WAF Cleanup Mod Security.")
}
