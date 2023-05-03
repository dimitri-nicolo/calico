// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package checker

import (
	"fmt"
	"strings"

	authz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"

	log "github.com/sirupsen/logrus"

	"google.golang.org/genproto/googleapis/rpc/status"

	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/app-policy/waf"
	"github.com/projectcalico/calico/felix/tproxydefs"
)

type WAFCheckProvider struct {
	subscriptionType string
	checkFn          func(req *authz.CheckRequest) (*authz.CheckResponse, error)
}

type WAFCheckProviderOption func(*WAFCheckProvider)

func WithWAFCheckProviderCheckFn(fn func(req *authz.CheckRequest) (*authz.CheckResponse, error)) WAFCheckProviderOption {
	return func(wp *WAFCheckProvider) {
		wp.checkFn = fn
	}
}

func NewWAFCheckProvider(subscriptionType string, opts ...WAFCheckProviderOption) CheckProvider {
	c := &WAFCheckProvider{
		subscriptionType: subscriptionType,
		checkFn:          defaultWAFCheck,
	}

	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *WAFCheckProvider) Name() string {
	return "web-application-firewall"
}

func (c *WAFCheckProvider) Check(ps *policystore.PolicyStore, req *authz.CheckRequest) (*authz.CheckResponse, error) {
	switch c.subscriptionType {
	case "per-host-policies":
		if wafIPSet, ok := ps.IPSetByID[tproxydefs.ServiceIPsIPSet]; ok &&
			wafIPSet.ContainsAddress(req.Attributes.Destination.Address) {
			return c.checkFn(req)
		}

		// traffic described in request doesn't need to go through WAF check; or
		// traffic described in request is plaintext;
		// or sent here by mistake.
		//
		// in any case, let it continue to next check
		return &authz.CheckResponse{Status: &status.Status{Code: UNKNOWN}}, nil
	default:
		return c.checkFn(req)
	}
}

func defaultWAFCheck(req *authz.CheckRequest) (*authz.CheckResponse, error) {
	resp := &authz.CheckResponse{Status: &status.Status{Code: OK}}

	// Helper variables used to reduce potential code smells.
	reqMethod := req.GetAttributes().GetRequest().GetHttp().GetMethod()
	reqPath := req.GetAttributes().GetRequest().GetHttp().GetPath()
	reqHost := req.GetAttributes().GetRequest().GetHttp().GetHost()
	reqProtocol := req.GetAttributes().GetRequest().GetHttp().GetProtocol()
	reqSourceHost := req.GetAttributes().GetSource().GetAddress().GetSocketAddress().GetAddress()
	reqSourcePort := req.GetAttributes().GetSource().GetAddress().GetSocketAddress().GetPortValue()
	reqDestinationHost := req.GetAttributes().GetDestination().GetAddress().GetSocketAddress().GetAddress()
	reqDestinationPort := req.GetAttributes().GetDestination().GetAddress().GetSocketAddress().GetPortValue()
	reqHeaders := req.GetAttributes().GetRequest().GetHttp().GetHeaders()
	reqBody := req.GetAttributes().GetRequest().GetHttp().GetBody()

	// WAF ModSecurity Process Http Request.
	err := WafProcessHttpRequest(
		reqPath, reqMethod, reqProtocol, reqSourceHost,
		reqSourcePort, reqDestinationHost, reqDestinationPort,
		reqHost, reqHeaders, reqBody,
	)
	if err != nil {
		log.Errorf("WAF Process Http Request URL '%s' WAF rules rejected HTTP request!", reqPath)
		resp.Status.Code = PERMISSION_DENIED
		return resp, nil
	}

	return resp, nil
}

func WafProcessHttpRequest(uri, httpMethod, inputProtocol, clientHost string, clientPort uint32, serverHost string, serverPort uint32, destinationHost string, reqHeaders map[string]string, reqBody string) error {

	// Use this as the correlationID.
	id := waf.GenerateModSecurityID()

	httpProtocol, httpVersion := splitInput(inputProtocol, "/", "HTTP", "1.1")
	err := waf.ProcessHttpRequest(id, uri, httpMethod, httpProtocol, httpVersion, clientHost, clientPort, serverHost, serverPort, reqHeaders, reqBody)

	// Collect OWASP log information:
	owaspLogInfo := waf.GetAndClearOwaspLogs(id)

	action := "pass-through"
	if err != nil {
		action = "blocked"
	}
	for _, owaspInfo := range owaspLogInfo {
		// Log to Elasticsearch => Kibana.
		waf.Logger.WithFields(log.Fields{
			"path":     uri,
			"method":   httpMethod,
			"protocol": inputProtocol,
			"source": log.Fields{
				"ip":       clientHost,
				"port_num": clientPort,
				"hostname": "-",
			},
			"destination": log.Fields{
				"ip":       serverHost,
				"port_num": serverPort,
				"hostname": destinationHost,
			},
			"rule_info": owaspInfo.String(),
		}).Error(
			fmt.Sprintf("[%s] %s", action, owaspInfo.Message),
		)

		// Log to Dikastes logs.
		log.WithFields(log.Fields{
			"id":      id,
			"url":     uri,
			"message": owaspInfo.Message,
		}).Warn(owaspInfo.String())
	}

	return err
}

// splitInput: split input based on delimiter specified into 2x components [left and right].
// if input cannot be split into 2x components based on delimiter then use default values specified.
// input example: "HTTP/1.1"
// output return: "HTTP" and "1.1"
func splitInput(input, delim, defaultLeft, defaultRight string) (actualLeft, actualRight string) {
	splitN := strings.SplitN(input, delim, 2)
	length := len(splitN)

	actualLeft = defaultLeft
	actualRight = defaultRight

	if length == 1 && len(splitN[0]) > 0 {
		actualLeft = splitN[0]
	}
	if length == 2 && len(splitN[1]) > 0 {
		actualRight = splitN[1]
	}

	return actualLeft, actualRight
}
