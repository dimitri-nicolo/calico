package waf

import (
	"strconv"
	"sync"

	corazatypes "github.com/corazawaf/coraza/v3/types"
	envoyauthz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/projectcalico/calico/felix/proto"
)

type wafEventsPipeline struct {
	mu            sync.Mutex
	errorsByTx    map[string][]corazatypes.MatchedRule
	flushCallback eventCallbackFn
}

func NewEventsPipeline(cb eventCallbackFn) *wafEventsPipeline {
	return &wafEventsPipeline{
		errorsByTx:    map[string][]corazatypes.MatchedRule{},
		flushCallback: cb,
	}
}

func (p *wafEventsPipeline) ProcessErrorRule(rule corazatypes.MatchedRule) {
	txID := rule.TransactionID()

	p.mu.Lock()
	defer p.mu.Unlock()

	txErrs := p.errorsByTx[txID]
	txErrs = append(txErrs, rule)
	p.errorsByTx[txID] = txErrs
}

// Process can take two types of events:
// - corazatypes.MatchedRule
// - *txHttpInfo
// We use the first type to process matched rules and create cache entries
// The second type is used to fill in missing info in the cache entries
//
// if the cached entries do not have the missing info yet and flush comes along
// that's okay, we'll just fill in the missing info with the default values
func (p *wafEventsPipeline) Process(checkReq *envoyauthz.CheckRequest, tx corazatypes.Transaction) {
	txID := tx.ID()

	p.mu.Lock()
	matchedRules, ok := p.errorsByTx[txID]
	if !ok {
		p.mu.Unlock()
		return
	}
	delete(p.errorsByTx, txID)
	p.mu.Unlock()

	log.WithField("rules", matchedRules).Debug("Processing matched rules")
	attr := checkReq.Attributes
	entry := &proto.WAFEvent{
		TxId:    txID,
		Host:    attr.Request.Http.Host,
		SrcIp:   attr.Source.Address.GetSocketAddress().Address,
		SrcPort: int32(attr.Source.Address.GetSocketAddress().GetPortValue()),
		DstIp:   attr.Destination.Address.GetSocketAddress().Address,
		DstPort: int32(attr.Destination.Address.GetSocketAddress().GetPortValue()),
		Rules:   []*proto.WAFRuleHit{},
	}

	req := attr.Request
	entry.Request = &proto.HTTPRequest{
		Method:  req.Http.Method,
		Path:    req.Http.Path,
		Version: req.Http.Protocol,
		Headers: req.Http.Headers,
	}
	entry.Timestamp = &timestamppb.Timestamp{
		Seconds: req.Time.Seconds,
		Nanos:   req.Time.Nanos,
	}
	entry.Action = "pass"
	if in := tx.Interruption(); in != nil {
		entry.Action = in.Action
	}

	for _, matchedRule := range matchedRules {
		rule := matchedRule.Rule()
		entry.Rules = append(entry.Rules, &proto.WAFRuleHit{
			Rule: &proto.WAFRule{
				Id:       strconv.Itoa(rule.ID()),
				Message:  matchedRule.Message(),
				Severity: rule.Severity().String(),
				File:     rule.File(),
				Line:     strconv.Itoa(rule.Line()),
			},
			Disruptive: matchedRule.Disruptive(),
		})
	}

	p.flushCallback(entry)
}
