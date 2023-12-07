// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package flags

import (
	"encoding/json"
	"flag"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	ListenNetwork     string      `json:"listenNetwork,omitempty"`
	ListenAddress     string      `json:"listenAddress,omitempty"`
	DialNetwork       string      `json:"dialNetwork,omitempty"`
	DialAddress       string      `json:"dialAddress,omitempty"`
	LogLevel          string      `json:"logLevel,omitempty"`
	WAFEnabled        bool        `json:"wafEnabled,omitempty"`
	WAFDirectives     stringArray `json:"wafDirectives,omitempty"`
	WAFRulesetBaseDir string      `json:"wafRulesetBaseDir,omitempty"`
	WAFLogFile        string      `json:"wafLogFile,omitempty"`
	SubscriptionType  string      `json:"subscriptionType,omitempty"`
	HTTPServerAddr    string      `json:"httpServerAddr,omitempty"`
	HTTPServerPort    string      `json:"httpServerPort,omitempty"`

	*flag.FlagSet `json:"-"`
}

func New() *Config {
	fs := flag.NewFlagSet("dikastes", flag.ExitOnError)

	cfg := &Config{
		FlagSet: fs,
	}

	fs.StringVar(&cfg.ListenAddress, "listen", "/var/run/dikastes/dikastes.sock", "Listen address")
	fs.StringVar(&cfg.ListenNetwork, "listen-network", "unix", "Listen network e.g. tcp, unix")
	fs.StringVar(&cfg.DialAddress, "dial", "/var/run/nodeagent/socket", "PolicySync address")
	fs.StringVar(&cfg.DialNetwork, "dial-network", "unix", "PolicySync network e.g. tcp, unix")
	fs.StringVar(&cfg.LogLevel, "log-level", "info", "Log at specified level e.g. panic, fatal,info, debug, trace")
	fs.BoolVar(&cfg.WAFEnabled, "waf-enabled", false, "Enable WAF.")
	fs.StringVar(&cfg.WAFRulesetBaseDir, "waf-ruleset-base-dir", "/etc/modsecurity-ruleset", "Base directory for WAF rulesets.")
	fs.Var(&cfg.WAFDirectives, "waf-directive", "Additional directives to specify for WAF (if enabled). Can be specified multiple times.")
	fs.StringVar(&cfg.WAFLogFile, "waf-log-file", "", "WAF log file path. e.g. /var/log/calico/waf/waf.log")
	fs.StringVar(&cfg.SubscriptionType,
		"subscription-type",
		getEnv("DIKASTES_SUBSCRIPTION_TYPE", "per-host-policies"),
		"Subscription type e.g. per-pod-policies, per-host-policies",
	)
	fs.StringVar(
		&cfg.HTTPServerAddr,
		"http-server-addr",
		getEnv("DIKASTES_HTTP_BIND_ADDR", "0.0.0.0"),
		"HTTP server address",
	)
	fs.StringVar(
		&cfg.HTTPServerPort,
		"http-server-port",
		getEnv("DIKASTES_HTTP_PORT", ""),
		"HTTP server port",
	)

	return cfg
}

func (c *Config) Parse(args []string) error {
	// we handle the presence of subcommands here
	// legacy arguments are:
	// - dikastes server -dial /var/run/nodeagent/nodeagent.sock -listen /var/run/dikastes/dikastes.sock
	// - dikastes client <namespace> <account> -dial /var/run/nodeagent/nodeagent.sock
	// new arguments are (preferred, client is now deprecated):
	// - dikastes --dial /var/run/nodeagent/nodeagent.sock --listen /var/run/dikastes/dikastes.sock

	switch {
	case len(args) < 2: // args[0] is program name, args[1] is subcommand
		return c.FlagSet.Parse(args) // handle no subcommand, no args
	case args[1] == "server": // handle with subcommand
		return c.FlagSet.Parse(args[2:])
	case args[1] == "client":
		os.Exit(1) // client is deprecated
	default: // all other cases
		return c.FlagSet.Parse(args[1:])
	}
	return nil
}

func (c *Config) Fields() log.Fields {
	b, err := json.Marshal(c)
	if err != nil {
		return log.Fields{}
	}
	var f log.Fields
	if err := json.Unmarshal(b, &f); err != nil {
		return log.Fields{}
	}
	return f
}

type stringArray []string

func (i *stringArray) String() string {
	return strings.Join(*i, ", ")
}

func (i *stringArray) Value() []string {
	return *i
}

func (i *stringArray) Set(value string) error {
	*i = append(*i, strings.Trim(value, "\"'"))
	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
