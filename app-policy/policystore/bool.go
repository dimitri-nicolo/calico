package policystore

import (
	"strconv"

	log "github.com/sirupsen/logrus"
)

func getBoolFromConfig(m map[string]string, name string, def bool) bool {
	b := def
	if v, ok := m[name]; ok {
		log.Debugf("%s is present in config", name)
		if p, err := strconv.ParseBool(v); err == nil {
			log.Debugf("Parsed value from Felix config: %s=%v", name, p)
			b = p
		} else {
			log.Errorf("Unknown %s boolean value: %s", name, v)
		}
	}
	return b
}
