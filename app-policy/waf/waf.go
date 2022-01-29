// Copyright (c) 2018-2022 Tigera, Inc. All rights reserved.

package waf

// #cgo CFLAGS: -I/usr/local/modsecurity/include
// #cgo LDFLAGS: -L/usr/local/modsecurity/lib/ -Wl,-rpath -Wl,/usr/local/modsecurity/lib/ -lmodsecurity
// #include "waf.h"
import "C"
import (
	"errors"
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

func ExtractRulesSetFilenames() ([]string, error) {
	// Read all core rule set file names from rules directory.
	var files []string
	items, err := ioutil.ReadDir(rulesetDirectory)
	if err != nil {
		return nil, err
	}

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
		log.Infof("WAF Found Rules File[%d]('%s')", count, file)
		count++
	}

	log.Infof("WAF Total Core Rules Set files: %d", len(files))
	return files, nil
}

func LoadModSecurityCoreRuleSet(filenames []string) int {

	size := len(filenames)
	load := 0

	log.Infof("WAF Attempt load %d Core Rule Set files", size)
	for _, filename := range filenames {
		success := loadModSecurityCoreRuleSetImpl(filename)
		if success {
			load++
		}
	}

	log.Infof("WAF Process load %d Core Rule Set files  SUCCESS", load)
	if size != load {
		log.Infof("WAF Process load %d Core Rule Set files  FAILURE", size-load)
	}

	return load
}
func loadModSecurityCoreRuleSetImpl(filename string) bool {

	// Assume core rule set file loads OK.
	success := true

	Cfilename := C.CString(filename)
	defer C.free(unsafe.Pointer(Cfilename))

	// Attempt to load core rule set file;
	// any error generated from ModSec will be returned directly.
	Cpayload := C.LoadModSecurityCoreRuleSet(Cfilename)
	if Cpayload != nil {
		errStr := C.GoString(Cpayload)
		C.free(unsafe.Pointer(Cpayload))

		if len(errStr) > 0 {
			log.Errorf("WAF Error attempt load file '%s' => '%v'", filename, errStr)
			success = false
		}
	}

	return success
}

func GenerateModSecurityID() string {
	return uuid.New().String()
}

func ProcessHttpRequest(id, url, httpMethod, httpProtocol, httpVersion string, clientHost string, clientPort uint32, serverHost string, serverPort uint32) error {
	prefix := getProcessHttpRequestPrefix(id)
	log.Printf("%s URL '%s'", prefix, url)

	Cid := C.CString(id)
	defer C.free(unsafe.Pointer(Cid))
	Curi := C.CString(url)
	defer C.free(unsafe.Pointer(Curi))
	ChttpMethod := C.CString(httpMethod)
	defer C.free(unsafe.Pointer(ChttpMethod))
	ChttpProtocol := C.CString(httpProtocol)
	defer C.free(unsafe.Pointer(ChttpProtocol))
	ChttpVersion := C.CString(httpVersion)
	defer C.free(unsafe.Pointer(ChttpVersion))
	CclientHost := C.CString(clientHost)
	defer C.free(unsafe.Pointer(CclientHost))
	CserverHost := C.CString(serverHost)
	defer C.free(unsafe.Pointer(CserverHost))
	CclientPort := C.int(clientPort)
	CserverPort := C.int(serverPort)

	start := time.Now()
	detection := int(C.ProcessHttpRequest(Cid, Curi, ChttpMethod, ChttpProtocol, ChttpVersion, CclientHost, CclientPort, CserverHost, CserverPort))
	elapsed := time.Since(start)

	log.Infof("%s URL '%s' Detection=%d Time elapsed: %s", prefix, url, detection, elapsed)
	if detection > 0 {
		errMsg := fmt.Sprintf("%s URL '%s' Detection=%d", prefix, url, detection)
		return errors.New(errMsg)
	}

	return nil
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
