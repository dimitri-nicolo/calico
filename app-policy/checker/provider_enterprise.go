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
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/felix/tproxydefs"
)

type wafCheckFn func(ps *policystore.PolicyStore, req *authz.CheckRequest, src, dst []proto.WorkloadEndpointID) (*authz.CheckResponse, error)

type WAFCheckProvider struct {
	subscriptionType string
	checkFn          wafCheckFn
}

type WAFCheckProviderOption func(*WAFCheckProvider)

func WithWAFCheckProviderCheckFn(fn wafCheckFn) WAFCheckProviderOption {
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
			// lookup endpoints see if we get src or dest
			src, dst, _ := lookupEndpointKeysFromRequest(ps, req)
			if len(dst) == 0 {
				// if we get to here, this means traffic was steered in from tproxy.
				// but, WAF check is being processed at source (has src info, no dest info)
				// continue to next (dest) node by returning OK (since waf is currently the last check performed )
				log.Debug("WAF Check encountered at source. continuing to next check")
				return &authz.CheckResponse{Status: &status.Status{Code: OK}}, nil
			}
			return c.checkFn(ps, req, src, dst)
		}

		// traffic described in request doesn't need to go through WAF check; or
		// traffic described in request is plaintext;
		// or sent here by mistake.
		//
		// in any case, let it continue to next check
		return &authz.CheckResponse{Status: &status.Status{Code: UNKNOWN}}, nil
	default:
		return c.checkFn(ps, req, nil, nil)
	}
}

func defaultWAFCheck(ps *policystore.PolicyStore, req *authz.CheckRequest, src, dst []proto.WorkloadEndpointID) (*authz.CheckResponse, error) {
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
		src, dst,
	)
	if err != nil {
		log.Errorf("WAF Process Http Request URL '%s' WAF rules rejected HTTP request!", reqPath)
		resp.Status.Code = PERMISSION_DENIED
		return resp, nil
	}

	return resp, nil
}

func WafProcessHttpRequest(
	uri, httpMethod, inputProtocol,
	clientHost string, clientPort uint32,
	serverHost string, serverPort uint32,
	destinationHost string,
	reqHeaders map[string]string,
	reqBody string,
	src, dst []proto.WorkloadEndpointID,
) error {

	// Use this as the correlationID.
	id := waf.GenerateModSecurityID()

	httpProtocol, httpVersion := splitInput(inputProtocol, "/", "HTTP", "1.1")
	// Fix request header Host
	if reqHeaders == nil {
		reqHeaders = map[string]string{}
	}
	reqHeaders["host"] = destinationHost
	err := waf.ProcessHttpRequest(id, uri, httpMethod, httpProtocol,
		httpVersion, clientHost, clientPort, serverHost, serverPort,
		reqHeaders, reqBody)

	// Collect OWASP log information:
	var owaspLogInfo []*waf.OwaspInfo
	owaspLogInfo = waf.GetAndClearOwaspLogs(id)

	action := "pass-through"
	if err != nil {
		action = "blocked"
		owaspInfo := waf.NewOwaspInfo(waf.ParseLog(err.(waf.WAFError).Disruption.Log))
		owaspInfo.Disruptive = true
		owaspLogInfo = append(owaspLogInfo, owaspInfo)
	}
	var rules []log.Fields
	var rule_info []string
	for _, owaspInfo := range owaspLogInfo {
		rules = append(rules, log.Fields{
			"id":         owaspInfo.RuleId,
			"message":    owaspInfo.Message,
			"severity":   owaspInfo.Severity,
			"file":       owaspInfo.File,
			"line":       owaspInfo.Line,
			"disruptive": owaspInfo.Disruptive,
		})

		// Log to Dikastes logs.
		log.WithFields(log.Fields{
			"id":      id,
			"url":     uri,
			"message": owaspInfo.Message,
		}).Warn(owaspInfo.String())
		rule_info = append(rule_info, owaspInfo.String())
	}

	srcNamespace, srcName := extractFirstWepNameAndNamespace(src)
	dstNamespace, dstName := extractFirstWepNameAndNamespace(dst)

	if log.IsLevelEnabled(log.TraceLevel) {
		log.WithFields(log.Fields{
			"srcName":      srcName,
			"srcNamespace": srcNamespace,
			"dstName":      dstName,
			"dstNamespace": dstNamespace,
		}).Trace("logged names and namespaces")
	}

	if rules != nil {
		// Log to Elasticsearch => Kibana.
		waf.Logger.WithFields(log.Fields{
			"request_id": id,
			"path":       uri,
			"method":     httpMethod,
			"protocol":   inputProtocol,
			"source": log.Fields{
				"ip":        clientHost,
				"port_num":  clientPort,
				"hostname":  "-",
				"name":      srcName,
				"namespace": srcNamespace,
			},
			"destination": log.Fields{
				"ip":        serverHost,
				"port_num":  serverPort,
				"hostname":  destinationHost,
				"name":      dstName,
				"namespace": dstNamespace,
			},
			// keeping this field only for backward compatibility
			"rule_info": strings.Join(rule_info, "\n"),
			"rules":     rules,
		}).Error(
			fmt.Sprintf("[%s] %d WAF rule(s) got hit", action, len(rules)),
		)
	}

	return err
}

func extractFirstWepNameAndNamespace(weps []proto.WorkloadEndpointID) (string, string) {
	if len(weps) == 0 {
		return "-", "-"
	}

	wepName := weps[0].WorkloadId
	parts := strings.Split(wepName, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return wepName, "-"
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
