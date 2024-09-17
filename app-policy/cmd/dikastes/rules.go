package main

import (
	"fmt"
)

const (
	inputRedirectChain = "CALII_PROXY_REDIRECT"
	inputProxyInbound  = "CALII_PROXY_INBOUND"
)

func generateRules(
	envoyInboundPort, envoyMetricsPort, envoyLivenessPort, envoyReadinessPort, envoyStartupProbePort, envoyHealthCheckPort string,
) (inboundStaticRules []ruleSpecSet) {
	inboundStaticRules = []ruleSpecSet{
		{
			table: "nat",
			chain: inputRedirectChain,
			ruleSpecs: []string{
				"-p", "tcp",
				"-j", "REDIRECT", "--to-port", envoyInboundPort,
				"-m", "comment", "--comment", "Redirect inbound traffic to envoy",
			},
		},
		{
			table: "nat",
			chain: "PREROUTING",
			ruleSpecs: []string{
				"-p", "tcp",
				"-j", inputProxyInbound,
				"-m", "comment", "--comment", fmt.Sprintf("Jump to %s chain for ALL inbound traffic", inputProxyInbound),
			},
		},
		{
			table: "nat",
			chain: inputProxyInbound,
			ruleSpecs: []string{
				"-p", "tcp",
				"--dport", envoyMetricsPort,
				"-j", "RETURN",
				"-m", "comment", "--comment", "Allow access to envoy metrics port",
			},
		},
		{
			table: "nat",
			chain: inputProxyInbound,
			ruleSpecs: []string{
				"-p", "tcp",
				"--dport", envoyLivenessPort,
				"-j", "RETURN",
				"-m", "comment", "--comment", "Allow access to envoy liveness probe port",
			},
		},
		{
			table: "nat",
			chain: inputProxyInbound,
			ruleSpecs: []string{
				"-p", "tcp",
				"--dport", envoyReadinessPort,
				"-j", "RETURN",
				"-m", "comment", "--comment", "Allow access to envoy readiness probe port",
			},
		},
		{
			table: "nat",
			chain: inputProxyInbound,
			ruleSpecs: []string{
				"-p", "tcp",
				"--dport", envoyStartupProbePort,
				"-j", "RETURN",
				"-m", "comment", "--comment", "Allow access to envoy startup probe port",
			},
		},
		{
			table: "nat",
			chain: inputProxyInbound,
			ruleSpecs: []string{
				"-p", "tcp",
				"--dport", envoyHealthCheckPort,
				"-j", "RETURN",
				"-m", "comment", "--comment", "Allow access to envoy health check port",
			},
		},
		{
			table: "nat",
			chain: inputProxyInbound,
			ruleSpecs: []string{
				"-p", "tcp",
				"-j", inputRedirectChain,
				"-m", "comment", "--comment", "Redirect remaining inbound traffic to envoy",
			},
		},
	}

	return inboundStaticRules
}
