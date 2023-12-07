// Copyright (c) 2023 Tigera, Inc. All rights reserved.
// Package waf implements a coraza-based checker.CheckProvider server.
// In this case, this authorization server provides a WAF service for the external authz request.
//
// The WAF service is implemented as a checker.CheckProvider for the checker.Checker server.
// The checker.Checker server is a gRPC server that implements envoy ext_authz
package waf

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	coreruleset "github.com/corazawaf/coraza-coreruleset"
	coraza "github.com/corazawaf/coraza/v3"
	corazatypes "github.com/corazawaf/coraza/v3/types"

	mergefs "github.com/jcchavezs/mergefs"

	code "google.golang.org/genproto/googleapis/rpc/code"
	status "google.golang.org/genproto/googleapis/rpc/status"

	envoycore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoyauthz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"

	"github.com/projectcalico/calico/app-policy/checker"
	"github.com/projectcalico/calico/app-policy/internal/util/io"
	"github.com/projectcalico/calico/app-policy/policystore"
	"github.com/projectcalico/calico/app-policy/types"
	"github.com/projectcalico/calico/felix/proto"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

var (
	OK   = newResponseWithCode(code.Code_OK, "OK")
	DENY = newResponseWithCode(code.Code_PERMISSION_DENIED, "Forbidden")
)

func newResponseWithCode(code code.Code, message string) *envoyauthz.CheckResponse {
	return &envoyauthz.CheckResponse{Status: &status.Status{Code: int32(code), Message: message}}
}

type eventCallbackFn func(interface{})

var _ checker.CheckProvider = (*Server)(nil)

type Server struct {
	coraza.WAF
	eventCallbacks []eventCallbackFn
}

func New(directives []string, rulesetBaseDir string, eventCallbacks ...eventCallbackFn) (*Server, error) {
	osFS := io.DirFS(rulesetBaseDir)
	cfg := coraza.NewWAFConfig().
		WithRootFS(mergefs.Merge(
			coreruleset.FS,
			osFS,
		))

	for _, d := range directives {
		cfg = cfg.WithDirectives(d)
	}

	waf, err := coraza.NewWAF(cfg)
	if err != nil {
		return nil, err
	}

	return &Server{
		WAF:            waf,
		eventCallbacks: eventCallbacks,
	}, nil
}

func (w *Server) Name() string {
	return "coraza"
}

func (w *Server) Check(st *policystore.PolicyStore, checkReq *envoyauthz.CheckRequest) (*envoyauthz.CheckResponse, error) {
	if log.IsLevelEnabled(log.DebugLevel) {
		log.WithFields(log.Fields{"attributes": checkReq.Attributes}).Debug("check request received")
	}

	attrs := checkReq.Attributes
	req := attrs.Request
	httpReq := req.Http

	tx := w.NewTransactionWithID(httpReq.Id)
	defer tx.Close()

	if tx.IsRuleEngineOff() {
		return OK, nil
	}

	dstHost, dstPort, _ := peerToHostPort(attrs.Destination)
	srcHost, srcPort, _ := peerToHostPort(attrs.Source)

	tx.ProcessConnection(srcHost, int(srcPort), dstHost, int(dstPort))
	tx.ProcessURI(httpReq.Path, httpReq.Method, httpReq.Protocol)
	for k, v := range httpReq.Headers {
		tx.AddRequestHeader(k, v)
	}
	if host := req.Http.Host; host != "" {
		tx.AddRequestHeader("Host", host)
		tx.SetServerName(host)
	}
	if it := tx.ProcessRequestHeaders(); it != nil {
		return w.processInterruption(st, checkReq, tx, it)
	}

	if tx.IsRequestBodyAccessible() {
		s := strings.NewReader(httpReq.Body)
		switch it, _, err := tx.ReadRequestBodyFrom(s); {
		case err != nil:
			return nil, err
		case it != nil:
		}
	}
	switch it, err := tx.ProcessRequestBody(); {
	case err != nil:
		return nil, err
	case it != nil:
		return w.processInterruption(st, checkReq, tx, it)
	}
	return OK, nil
}

func (w *Server) processInterruption(st *policystore.PolicyStore, checkReq *envoyauthz.CheckRequest, tx corazatypes.Transaction, it *corazatypes.Interruption) (*envoyauthz.CheckResponse, error) {
	resp := &envoyauthz.CheckResponse{Status: &status.Status{Code: int32(code.Code_OK)}}
	defer w.auditEventsFromTransactionInterruption(st, checkReq, tx, it)

	switch it.Action {
	// We only handle disruptive actions here.
	// However, not all disruptive actions mean a change in response code. See below:
	// - drop Initiates an immediate close of the TCP connection by sending a FIN packet.
	// - deny Stops rule processing and intercepts transaction.
	// - block Performs the disruptive action defined by the previous SecDefaultAction.
	// - pause Pauses transaction processing for the specified number of milliseconds. We don't support this, yet.
	// - proxy Intercepts the current transaction by forwarding the request to another web server using the proxy backend.
	// 		The forwarding is carried out transparently to the HTTP client (i.e., there’s no external redirection taking place)
	// - redirect Intercepts transaction by issuing an external (client-visible) redirection to the given location
	//
	// for more info about actions: https://coraza.io/docs/seclang/actions/ and note the Action Group for each.
	case "allow":
		// default response code is OK, do nothing but return OK
	case "drop", "deny", "block":
		resp = newResponseWithCode(
			statusFromAction(it, code.Code_PERMISSION_DENIED),
			messageFromInterruption(it),
		)
	case "pause", "proxy", "redirect":
		log.Warnf("unsupported action (%s), proceeding with no-op", it.Action)
	default:
		// all other actions should be non-disruptive. Do nothing but return OK
	}

	return resp, nil
}

func (w *Server) auditEventsFromTransactionInterruption(st *policystore.PolicyStore, checkReq *envoyauthz.CheckRequest, tx corazatypes.Transaction, it *corazatypes.Interruption) {
	rules := corazaRulesToLinseedWAFRuleHit(tx)
	src, dst, err := checker.LookupEndpointKeysFromRequest(st, checkReq)
	switch err.(type) {
	case types.ErrNoStore, types.ErrUnprocessable:
		// if there's no store or if endpoint info is generally unprocessable,
		// we'll have no ability to lookup the endpoint information.
		// in this case, just log a warning and continue as it is not a fatal error.
		if log.IsLevelEnabled(log.DebugLevel) {
			log.WithError(err).Debug("error interacting with store; no wep info for audit event")
		}
	case nil:
		// do nothing
	default:
		log.WithError(err).Panic("encountered unexpected error interacting with store")
	}
	srcNamespace, srcName := extractFirstWepNameAndNamespace(src)
	dstNamespace, dstName := extractFirstWepNameAndNamespace(dst)
	dstAddr, dstPort := envoyCoreAddressToIPPort(checkReq.Attributes.Destination.Address)
	srcAddr, srcPort := envoyCoreAddressToIPPort(checkReq.Attributes.Source.Address)

	wafLog := &v1.WAFLog{
		Timestamp: time.Now(),
		Level:     "WARN",
		RequestId: checkReq.Attributes.Request.Http.Id,
		Method:    checkReq.Attributes.Request.Http.Method,
		Path:      checkReq.Attributes.Request.Http.Path,
		Protocol:  checkReq.Attributes.Request.Http.Protocol,
		Msg:       messageFromInterruption(it),
		Rules:     rules,
		Destination: &v1.WAFEndpoint{
			IP:           dstAddr,
			PortNum:      int32(dstPort),
			Hostname:     checkReq.Attributes.Request.Http.Host,
			PodName:      dstName,
			PodNameSpace: dstNamespace,
		},
		Source: &v1.WAFEndpoint{
			IP:           srcAddr,
			PortNum:      int32(srcPort),
			PodName:      srcName,
			PodNameSpace: srcNamespace,
		},
		Host: checkReq.Attributes.Request.Http.Host,
	}
	w.runCallbacks(wafLog)
}

func (w *Server) runCallbacks(wafLog *v1.WAFLog) {
	for _, cb := range w.eventCallbacks {
		cb(wafLog)
	}
}

func messageFromInterruption(it *corazatypes.Interruption) string {
	return fmt.Sprintf("WAF rule %d interrupting request: %s (%d)", it.RuleID, it.Action, it.Status)
}

func statusFromAction(it *corazatypes.Interruption, fallbackValue code.Code) code.Code {
	if it.Status != 0 {
		return statusToCode(it.Status)
	}
	return fallbackValue
}

func statusToCode(s int) code.Code {
	switch s {
	case http.StatusOK:
		return code.Code_OK
	case http.StatusBadRequest:
		return code.Code_INVALID_ARGUMENT
	case http.StatusUnauthorized:
		return code.Code_UNAUTHENTICATED
	case http.StatusForbidden:
		return code.Code_PERMISSION_DENIED
	case http.StatusNotFound:
		return code.Code_NOT_FOUND
	case http.StatusConflict:
		return code.Code_ALREADY_EXISTS
	case http.StatusTooManyRequests:
		return code.Code_RESOURCE_EXHAUSTED
	case 499: // HTTP 499 Client Closed Request
		return code.Code_CANCELLED
	case http.StatusInternalServerError:
		return code.Code_INTERNAL
	case http.StatusNotImplemented:
		return code.Code_UNIMPLEMENTED
	case http.StatusBadGateway:
		return code.Code_UNAVAILABLE
	case http.StatusServiceUnavailable:
		return code.Code_UNAVAILABLE
	case http.StatusGatewayTimeout:
		return code.Code_DEADLINE_EXCEEDED
	}
	return code.Code_UNKNOWN
}

func peerToHostPort(peer *envoyauthz.AttributeContext_Peer) (host string, port uint32, ok bool) {
	switch v := peer.Address.Address.(type) {
	case *envoycore.Address_SocketAddress:
		host = v.SocketAddress.Address
		switch vv := v.SocketAddress.PortSpecifier.(type) {
		case *envoycore.SocketAddress_PortValue:
			port = vv.PortValue
			ok = true
		}
		return
	}
	return "127.0.0.1", 80, false
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

func corazaRulesToLinseedWAFRuleHit(tx corazatypes.Transaction) (hits []v1.WAFRuleHit) {
	for _, matchedRule := range tx.MatchedRules() {
		ruleInfo := matchedRule.Rule()
		hits = append(hits, v1.WAFRuleHit{
			Message:    matchedRule.Message(),
			Disruptive: matchedRule.Disruptive(),
			Id:         fmt.Sprint(ruleInfo.ID()),
			Severity:   ruleInfo.Severity().String(),
			File:       ruleInfo.File(),
			Line:       fmt.Sprint(ruleInfo.Line()),
		})
	}
	return
}

func envoyCoreAddressToIPPort(addr *envoycore.Address) (string, int64) {
	switch v := addr.Address.(type) {
	case *envoycore.Address_SocketAddress:
		return v.SocketAddress.Address, int64(v.SocketAddress.GetPortValue())
	case *envoycore.Address_Pipe:
		return v.Pipe.Path, 0
	case *envoycore.Address_EnvoyInternalAddress:
		return v.EnvoyInternalAddress.EndpointId, 0
	}
	return "", 0
}
