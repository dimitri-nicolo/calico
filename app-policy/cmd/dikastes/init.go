package main

import (
	"html/template"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/coreos/go-iptables/iptables"

	envoyconfig "github.com/projectcalico/calico/app-policy/envoy/config"
	"github.com/projectcalico/calico/app-policy/flags"
)

func runInit(config *flags.Config) {
	natv4, err := iptables.New()
	if err != nil {
		log.Fatal(err)
	}

	_ = natv4.NewChain("nat", inputRedirectChain)
	_ = natv4.NewChain("nat", inputProxyInbound)

	inboundStaticRules := generateRules(
		config.EnvoyInboundPort,
		config.EnvoyMetricsPort,
		config.EnvoyLivenessPort,
		config.EnvoyReadinessPort,
		config.EnvoyStartupProbePort,
		config.EnvoyHealthCheckPort,
	)
	for _, rule := range inboundStaticRules {
		if err := natv4.Append(rule.table, rule.chain, rule.ruleSpecs...); err != nil {
			log.
				WithFields(log.Fields{
					"table": rule.table,
					"chain": rule.chain,
					"rule":  strings.Join(rule.ruleSpecs, " "),
				}).
				WithError(err).
				Fatal("failed to add rule")
		}
	}

	// Save envoy-config file
	tpl := template.Must(template.New("envoy-config").
		Parse(envoyconfig.Config))
	envFile, err := os.Create(envoyconfig.Path)
	if err != nil {
		log.WithError(err).Fatal("Can't create envoy-config file")
	}
	err = tpl.Execute(envFile, config)
	if err != nil {
		log.WithError(err).Fatal("Error while processing envoy-config file")
	}
	err = envFile.Close()
	if err != nil {
		log.Fatal("Error while saving envoy-config file")
	}
}
