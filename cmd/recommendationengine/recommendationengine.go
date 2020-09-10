package main

import (
	"flag"

	log "github.com/sirupsen/logrus"

	"k8s.io/klog"

	"github.com/tigera/honeypod-recommendation/pkg/rule"
)

// const (
// 	// The health name for the reporter component.
// 	healthReporterName = "compliance-reporter"
// )

// var ver bool

func init() {
	// Tell klog to log into STDERR.
	var sflags flag.FlagSet
	klog.InitFlags(&sflags)
	err := sflags.Set("logtostderr", "true")
	if err != nil {
		log.WithError(err).Fatal("Failed to set logging configuration")
	}

	// // Add a flag to check the version.
	// flag.BoolVar(&ver, "version", false, "Print version information")
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "./config.yaml", "Config file path")
	flag.Parse()

	// log.SetLevel(log.InfoLevel)
	log.SetLevel(log.DebugLevel)

	config := parseConfig(configPath)

	// Generate global views
	globalRe := rule.NewRuleExecutor(config.GlobalRule, nil)
	if _, err := globalRe.Run(); err == nil {
		log.Debug("Global views generated")
	} else {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error generating global views")
	}

	for _, r := range config.Rules {
		re := rule.NewRuleExecutor(r, globalRe)
		recs, err := re.Run()
		if err == nil {
			log.WithFields(log.Fields{
				"rule":            r.Name,
				"recommendations": recs,
			}).Debug("Generator completed")
		} else {
			log.WithFields(log.Fields{
				"rule":  r.Name,
				"error": err,
			}).Error("Error executing rule")
		}
	}
}
