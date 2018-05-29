// Copyright (c) 2016-2018 Tigera, Inc. All rights reserved.
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

package ipsec

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	charonConfigRootDir   = "/etc/strongswan.d"
	charonMainConfigFile  = "charon.conf"
	charonFelixConfigFile = "set-by-felix.conf"

	charonConfigItemStdoutLogLevel = "charon.filelog.stdout.default"
	charonConfigItemStderrLogLevel = "charon.filelog.stderr.default"
)

var (
	// https://wiki.strongswan.org/projects/strongswan/wiki/LoggerConfiguration
	felixLogLevelToCharonLogLevel = map[string]string{
		"NONE": "-1",
		"NOTICE": "0",
		"INFO": "1",
		"DEBUG": "2",
		"VERBOSE": "4",
	}
)

// Data structure to handle charon config.
// https://wiki.strongswan.org/projects/strongswan/wiki/StrongswanConf
// Each section has a name, followed by C-Style curly brackets defining the sections body.
// Each section body contains a set of subsections and key/value pairs
//        settings := (section|keyvalue)*
//        section  := name { settings }
//        keyvalue := key = value\n

type configTree struct {
	section map[string]*configTree
	kv      map[string]string
}

func newConfigTree(items map[string]string) *configTree {
	tree := &configTree{}
	for k, v := range items {
		if err := tree.AddOneKV(k, v); err != nil {
			log.WithFields(log.Fields{
				"key":   k,
				"error": err,
			}).Error("Failed to add key to config tree.")
		}
	}

	return tree
}

// Add a dot notation kv pair to config tree.
func (t *configTree) AddOneKV(key, value string) error {
	// Breakdown key name into section slice and the real key.
	slice := strings.Split(key, ".")
	length := len(slice)
	if length <= 2 {
		// No dot in key name
		return fmt.Errorf("No dot in key name for configTree")
	}
	realKey := slice[length-1]
	sections := slice[:length-1]

	// Walk through configTree, create new section if necessary.
	currentSection := t
	for _, sectionName := range sections {
		nextSection, ok := currentSection.section[sectionName]
		if !ok {
			// Add or create a new section inside current section.
			// Make next section point to it.
			if currentSection.section == nil {
				currentSection.section = map[string]*configTree{sectionName: &configTree{}}
			} else {
				currentSection.section[sectionName] = &configTree{}
			}
			nextSection = currentSection.section[sectionName]
		}
		currentSection = nextSection
	}

	// Create or add new kv onto section.
	if currentSection.kv == nil {
		currentSection.kv = map[string]string{realKey: value}
	} else {
		currentSection.kv[realKey] = value
	}
	return nil
}

// Render configTree to strongswan config file format.
// StartSection: the section name to start with.
// linePrefix: the prefix for each line to indent. Normally it is couple of spaces.
// Result of a configTree with "charon.filelog.stdout.default": "2",
//                             "charon.filelog.stderr.default": "2",
//                             "charon.filelog.stderr.time_format": "%e %b %F"
//
//  charon {
//    filelog {
//      stdout {
//        default = 2
//      }
//      stderr {
//        default = 2
//        timestamp = %e %b %F
//      }
//    }
//  }
func (c *configTree) render(startSection, linePrefix string) string {
	var result string

	if startSection != "" {
		// Add indent for all except start of the tree.
		linePrefix += "  "
	}
	for k, v := range c.section {
		if v != nil {
			sectionHead := fmt.Sprintf("%s%s {\n", linePrefix, k)
			sectionBody := v.render(k, linePrefix)
			sectionEnd := fmt.Sprintf("%s}\n", linePrefix)
			result += sectionHead + sectionBody + sectionEnd
		}
	}

	for k, v := range c.kv {
		result += fmt.Sprintf("  %s%s = %s\n", linePrefix, k, v)
	}
	return result
}

// Structure to hold current charon config.
// We use dot notation for each config item, same as strongswan config doc.
// e.g. charon.filelog.stderr.default = 2
type CharonConfig struct {
	configFile string            // config file name
	items      map[string]string // dot notation key
}

func newCharonConfig(configFile string) *CharonConfig {
	// Initialise an empty felix config file.
	felixConfig := path.Join(charonConfigRootDir, configFile)
	panicIfErr(writeStringToFile(felixConfig, " "))

	// Insert felix config file into main charon config file.
	mainConfig := path.Join(charonConfigRootDir, charonMainConfigFile)
	panicIfErr(appendStringToFile(mainConfig, fmt.Sprintf("include %s\n", felixConfig)))

	return &CharonConfig{
		configFile: configFile,
		items:      map[string]string{},
	}
}

// Add configuration kv pairs to current charon config.
// The old value will be overwritten.
func (c *CharonConfig) AddKVs(kv map[string]string) {
	for k, v := range kv {
		c.items[k] = v
	}
}

func (c *CharonConfig) renderToString() string {
	return newConfigTree(c.items).render("", "")
}

// Render current charon config to config file.
func (c *CharonConfig) RenderToFile() {
	config := path.Join(charonConfigRootDir, c.configFile)
	panicIfErr(writeStringToFile(config, c.renderToString()))
}

func (c *CharonConfig) SetLogLevel(felixLevel string) {
	charonLevel := felixLogLevelToCharonLogLevel[strings.ToUpper(felixLevel)]
	c.AddKVs(map[string]string{
		charonConfigItemStderrLogLevel: charonLevel,
		charonConfigItemStdoutLogLevel: charonLevel,
		"charon.filelog.stdout.time_format": "%b %e %T", //This is for test purpose for now. Will remove it later.
	})
}

func writeStringToFile(path, text string) error {
	if err := ioutil.WriteFile(path, []byte(text), 0600); err != nil {
		return fmt.Errorf("Failed to write file %s", path)
	}
	return nil
}

func appendStringToFile(path, text string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(text)
	if err != nil {
		return err
	}
	return nil
}
