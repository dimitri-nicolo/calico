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
	"sync"
	"time"
	"unsafe"

	"github.com/google/uuid"

	log "github.com/sirupsen/logrus"
)

// WAF enabled flag accessible outside package via waf.IsEnabled()
var wafIsEnabled bool

// Directory where the Core Rules Set are stored.
var rulesetDirectory string

// Slice of filenames read from Core Rules Set directory.
var filenames []string

var owaspLogInfo = map[string][]map[string]string{}
var owaspLogMutex sync.Mutex

// CheckRulesSetExists
// invoke this WAF function first checking if rules argument set and if so with destination directory.
// if this directory does not exist OR zero *.conf Core Rule Sets files exist then do not enable WAF.
func CheckRulesSetExists(directory string) error {

	// Assume WAF is not enabled by default.
	wafIsEnabled = false

	DefineRulesSetDirectory(directory)
	err := CheckRulesSetDirectoryExists()
	if err != nil {
		return err
	}

	err = ExtractRulesSetFilenames()
	if err != nil {
		return err
	}

	wafIsEnabled = len(filenames) > 0
	return nil
}

func DefineRulesSetDirectory(directory string) {

	rulesetDirectory = directory
	log.Printf("WAF Core Rules Set directory: '%s'", rulesetDirectory)

	// Ensure rules directory ends with trailing slash.
	if !strings.HasSuffix(rulesetDirectory, "/") {
		rulesetDirectory = rulesetDirectory + "/"
	}
}

func CheckRulesSetDirectoryExists() error {

	_, err := os.Stat(rulesetDirectory)
	if os.IsNotExist(err) {
		return err
	}

	return nil
}

func ExtractRulesSetFilenames() error {

	// Read all core rule set file names from rules directory.
	items, err := ioutil.ReadDir(rulesetDirectory)
	if err != nil {
		return err
	}

	// Sort files descending to ensure lower cased files like crs-setup.conf are loaded first.
	// This is a requirement for Core Rules Set and REQUEST-901-INITIALIZATION.conf bootstrap.
	sortFileNameDescend(items)

	count := 1
	filenames = nil
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
		filenames = append(filenames, file)
		log.Infof("WAF Found Rules File[%d]('%s')", count, file)
		count++
	}

	if len(filenames) == 0 {
		errMsg := fmt.Sprintf("No WAF Core Rules Set found in '%s'", rulesetDirectory)
		return errors.New(errMsg)
	}

	log.Infof("WAF Total Core Rules Set files: %d", len(filenames))
	return nil
}

func InitializeModSecurity() {
	log.Printf("WAF Initialize Mod Security.")
	C.InitializeModSecurity()
}

func LoadModSecurityCoreRuleSet(filenames []string) error {

	size := len(filenames)

	log.Infof("WAF Attempt load %d Core Rule Set files", size)
	for _, filename := range filenames {
		err := loadModSecurityCoreRuleSetImpl(filename)
		if err != nil {
			return err
		}
	}

	log.Infof("WAF Process load %d Core Rule Set files  SUCCESS", size)
	return nil
}
func loadModSecurityCoreRuleSetImpl(filename string) error {

	Cfilename := C.CString(filename)
	defer C.free(unsafe.Pointer(Cfilename))

	// Attempt to load core rule set file;
	// any error generated from ModSec will be returned directly.
	Cpayload := C.LoadModSecurityCoreRuleSet(Cfilename)
	if Cpayload != nil {
		errStr := C.GoString(Cpayload)
		C.free(unsafe.Pointer(Cpayload))

		if len(errStr) > 0 {
			errMsg := fmt.Sprintf("WAF Error attempt load file '%s' => '%v'", filename, errStr)
			return errors.New(errMsg)
		}
	}

	return nil
}

func GenerateModSecurityID() string {
	return uuid.New().String()
}

func ProcessHttpRequest(id, url, httpMethod, httpProtocol, httpVersion, clientHost string, clientPort uint32, serverHost string, serverPort uint32, reqHeaders map[string]string, reqBody string) error {
	prefix := GetProcessHttpRequestPrefix(id)

	message := fmt.Sprintf("%s URL '%v' Method '%s'", prefix, url, httpMethod)
	if len(reqBody) > 0 {
		message += fmt.Sprintf(" Body '%s'", reqBody)
	}
	log.Info(message)

	Cid := C.CString(id)
	defer C.free(unsafe.Pointer(Cid))

	// Request info.
	Curi := C.CString(url)
	defer C.free(unsafe.Pointer(Curi))
	ChttpMethod := C.CString(httpMethod)
	defer C.free(unsafe.Pointer(ChttpMethod))
	ChttpProtocol := C.CString(httpProtocol)
	defer C.free(unsafe.Pointer(ChttpProtocol))
	ChttpVersion := C.CString(httpVersion)
	defer C.free(unsafe.Pointer(ChttpVersion))

	// Request connection.
	CclientHost := C.CString(clientHost)
	defer C.free(unsafe.Pointer(CclientHost))
	CserverHost := C.CString(serverHost)
	defer C.free(unsafe.Pointer(CserverHost))
	CclientPort := C.int(clientPort)
	CserverPort := C.int(serverPort)

	// Request headers.
	reqHeaderSize := len(reqHeaders)
	CreqHeaderSize := C.int(reqHeaderSize)
	CreqHeaderKeys := C.makeCharArray(CreqHeaderSize)
	defer C.freeCharArray(CreqHeaderKeys, CreqHeaderSize)
	CreqHeaderVals := C.makeCharArray(CreqHeaderSize)
	defer C.freeCharArray(CreqHeaderVals, CreqHeaderSize)

	var index C.int
	for k, v := range reqHeaders {
		C.setArrayString(CreqHeaderKeys, C.CString(k), index)
		C.setArrayString(CreqHeaderVals, C.CString(v), index)
		index++
	}

	// Request body.
	reqBodySize := len(reqBody)
	CreqBodySize := C.int(reqBodySize)
	CreqBodyText := C.CString(reqBody)
	defer C.free(unsafe.Pointer(CreqBodyText))

	start := time.Now()
	retVal, err := C.ProcessHttpRequest(
		Cid,
		Curi,
		ChttpMethod,
		ChttpProtocol,
		ChttpVersion,
		CclientHost,
		CclientPort,
		CserverHost,
		CserverPort,
		CreqHeaderKeys,
		CreqHeaderVals,
		CreqHeaderSize,
		CreqBodyText,
		CreqBodySize)

	elapsed := time.Since(start)
	if err != nil {
		errMsg := fmt.Sprintf("%s URL '%s' ModSecurity error '%v'", prefix, url, err.Error())
		return errors.New(errMsg)
	}

	detection := int(retVal)
	log.Infof("%s URL '%s' Detection=%d Time elapsed: %s", prefix, url, detection, elapsed)

	if detection > 0 {
		errMsg := fmt.Sprintf("%s URL '%s' Detection=%d", prefix, url, detection)
		return errors.New(errMsg)
	}

	return nil
}

// IsEnabled helper function used by client calling code.
func IsEnabled() bool {
	return wafIsEnabled
}

// GetRulesDirectory public helper function for testing.
func GetRulesDirectory() string {
	return rulesetDirectory
}

// GetRulesSetFilenames helper function used for unit tests.
func GetRulesSetFilenames() []string {
	return filenames
}

func GetProcessHttpRequestPrefix(id string) string {
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
	uniqueId := dictionary[ParserUniqueId]

	// Ignore edge case in which log payload is missing the UniqueId;
	// in theory log should always include UniqueId because we construct ModSec transaction with this UniqueId explicitly.
	if len(uniqueId) == 0 {
		return
	}

	owaspLogMutex.Lock()
	defer owaspLogMutex.Unlock()

	owaspLogInfo[uniqueId] = append(owaspLogInfo[uniqueId], dictionary)
}

func GetAndClearOwaspLogs(uniqueId string) []string {

	owaspLogMutex.Lock()
	owaspInfos := owaspLogInfo[uniqueId]
	delete(owaspLogInfo, uniqueId)
	owaspLogMutex.Unlock()

	index := 1
	var owaspLogEntries []string
	for _, owaspDictionary := range owaspInfos {

		owaspFlattenLog := FormatMap(owaspDictionary)
		owaspLogEntries = append(owaspLogEntries, fmt.Sprintf("[%d] %s", index, owaspFlattenLog))
		index++
	}

	return owaspLogEntries
}

func CleanupModSecurity() {
	C.CleanupModSecurity()
	log.Printf("WAF Cleanup Mod Security.")
}
