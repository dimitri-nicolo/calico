package waf

import (
	"context"
	"fmt"
	"hash/crc64"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	corazatypes "github.com/corazawaf/coraza/v3/types"

	"github.com/projectcalico/calico/app-policy/checker"
	"github.com/projectcalico/calico/app-policy/policystore"
	linseedv1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

type cacheEntry struct {
	transactionID         string
	destIP, srcIP         string
	method, protocol      string
	host, path            string
	headers               map[string]string
	timestamp             time.Time
	srcPort, dstPort      uint32
	rules                 []linseedv1.WAFRuleHit
	count                 int
	action                string
	srcName, srcNamespace string
	finished              bool
}

type aggregationKey struct {
	method,
	host,
	path string
	rulesKey uint64
}

type wafEventsPipeline struct {
	cache         map[string]*cacheEntry
	aggregation   map[aggregationKey][]*cacheEntry
	aggMu         sync.Mutex
	flushCallback eventCallbackFn
	mu            sync.Mutex
	events        []interface{}

	currPolicyStore *policystore.PolicyStore
}

type txHttpInfo struct {
	txID, destIP, host     string
	path, method, protocol string
	headers                map[string]string
	timestamp              time.Time
	srcPort, dstPort       uint32
	action                 string
	srcName, srcNamespace  string
}

func NewEventsPipeline(cb eventCallbackFn) *wafEventsPipeline {
	return &wafEventsPipeline{
		cache:         make(map[string]*cacheEntry),
		aggregation:   map[aggregationKey][]*cacheEntry{},
		flushCallback: cb,
	}
}

// Process can take two types of events:
// - corazatypes.MatchedRule
// - *txHttpInfo
// We use the first type to process matched rules and create cache entries
// The second type is used to fill in missing info in the cache entries
//
// if the cached entries do not have the missing info yet and flush comes along
// that's okay, we'll just fill in the missing info with the default values
func (p *wafEventsPipeline) Process(ps *policystore.PolicyStore, event interface{}) {
	p.currPolicyStore = ps
	p.events = append(p.events, event)
}

func (p *wafEventsPipeline) processMatchedRule(matchedRule corazatypes.MatchedRule) {
	rule := matchedRule.Rule()
	if rule == nil {
		return
	}

	log.WithFields(log.Fields{
		"rule":  rule.ID(),
		"data":  matchedRule.Data(),
		"audit": matchedRule.AuditLog(),
	}).Debug("Processing matched rule")

	key := matchedRule.TransactionID()
	entry, ok := p.cache[key]
	if !ok {
		entry = &cacheEntry{
			transactionID: matchedRule.TransactionID(),
			destIP:        matchedRule.ServerIPAddress(),
			srcIP:         matchedRule.ClientIPAddress(),
			rules:         []linseedv1.WAFRuleHit{},
		}
		p.cache[key] = entry
	}

	entry.count++
	ruleInfo := matchedRule.Rule()
	entry.rules = append(entry.rules, linseedv1.WAFRuleHit{
		Message:    matchedRule.Message(),
		Disruptive: matchedRule.Disruptive(),
		Id:         fmt.Sprint(ruleInfo.ID()),
		Severity:   ruleInfo.Severity().String(),
		File:       ruleInfo.File(),
		Line:       fmt.Sprint(ruleInfo.Line()),
	})
}

func (p *wafEventsPipeline) processTxHttpInfo(info *txHttpInfo) {
	key := info.txID
	entry, ok := p.cache[key]
	if !ok {
		// only update the cache if we have a matching entry
		return
	}

	// fill in missing info
	entry.method = info.method
	entry.path = info.path
	entry.host = info.host
	entry.protocol = info.protocol
	entry.headers = info.headers
	entry.timestamp = info.timestamp
	entry.srcPort = info.srcPort
	entry.dstPort = info.dstPort
	entry.action = info.action
	entry.srcName = info.srcName
	entry.srcNamespace = info.srcNamespace
	// insert into aggregation map
	p.aggregationAdd(entry)
	// mark as finished transaction
	entry.finished = true
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func lookupSrcDstEps(entry *cacheEntry, ps *policystore.PolicyStore) (srcEp, dstEp *linseedv1.WAFEndpoint) {
	log.WithFields(log.Fields{
		"srcIP": entry.srcIP,
		"dstIP": entry.destIP,
	}).Debug("Looking up endpoint keys")
	src, dst, err := checker.LookupEndpointKeysFromSrcDst(
		ps,
		entry.srcIP,
		entry.destIP,
	)
	if err != nil {
		log.WithError(err).Debug("Failed to lookup endpoint keys")
		return
	}
	log.WithFields(log.Fields{
		"src": src,
		"dst": dst,
	}).Debug("Found endpoint keys")

	srcNamespaceFromCache, srcNameFromCache := extractFirstWepNameAndNamespace(src)
	dstNamespaceFromCache, dstNameFromCache := extractFirstWepNameAndNamespace(dst)

	// prioritize src info from the cache, then use the info from the entry.
	// use "-" for empty values as that is a special value for ES log entries
	srcName := firstNonEmpty(srcNameFromCache, entry.srcName, "-")
	srcNamespace := firstNonEmpty(srcNamespaceFromCache, entry.srcNamespace, "-")

	dstName := firstNonEmpty(dstNameFromCache, "-")
	dstNamespace := firstNonEmpty(dstNamespaceFromCache, "-")

	srcEp = &linseedv1.WAFEndpoint{
		IP:           entry.srcIP,
		PodName:      srcName,
		PodNameSpace: srcNamespace,
		PortNum:      int32(entry.srcPort),
	}

	dstEp = &linseedv1.WAFEndpoint{
		IP:           entry.destIP,
		PodName:      dstName,
		PodNameSpace: dstNamespace,
		PortNum:      int32(entry.dstPort),
	}
	return
}

func (p *wafEventsPipeline) cacheEntriesToLog(entries []*cacheEntry) *linseedv1.WAFLog {
	// XXX We are receiving the proper values here, but taking just the
	// first one for sending then through linseed.
	// This index should be restructured with a "Requests" field, that
	// proper bring the time and info of the others request fields of the
	// single requests that were resulting the same Threat
	entry := entries[0]

	srcEp, dstEp := lookupSrcDstEps(entry, p.currPolicyStore)

	log := &linseedv1.WAFLog{
		RequestId:   entry.transactionID,
		Source:      srcEp,
		Destination: dstEp,
		Rules:       entry.rules,
		Msg:         fmt.Sprintf("WAF detected %d violations %s", entry.count, bracket(entry.action)),
		// path needs to be aggregated in the future
		Path: entry.path,
		// we can't exactly source these from the matched rule alone
		// so we're hardcoding a value for now
		// these will get filled by processTxHttpInfo
		Method:   "GET",
		Protocol: "HTTP/1.1",
		Host:     "",
		// RuleInfo left blank because it's deprecated
	}

	return log
}

func bracket(s string) string {
	if s == "" {
		return s
	}
	return "[" + s + "]"
}

func (p *wafEventsPipeline) processEvents() {
	for _, event := range p.events {
		switch e := event.(type) {
		case corazatypes.MatchedRule:
			p.processMatchedRule(e)
		case *txHttpInfo:
			p.processTxHttpInfo(e)
		default:
			log.WithField("event", e).Warn("Unknown event type")
		}
	}
	p.events = nil
}

func (p *wafEventsPipeline) Flush() {
	// Gets the finished events on mutex and Unlock
	p.mu.Lock()
	finished := map[string]*cacheEntry{}

	p.processEvents()
	for k, v := range p.cache {
		if v.finished {
			finished[k] = v
			delete(p.cache, k)
		}
	}
	defer p.mu.Unlock()

	// removes finished ones for each aggregation key on mutex and Unlock
	readyAggregation := [][]*cacheEntry{}
	p.aggMu.Lock()
	for k, v := range p.aggregation {
		ready := []*cacheEntry{}
		unfinished := []*cacheEntry{}
		for _, v := range v {
			if _, ok := finished[v.transactionID]; ok {
				ready = append(ready, v)
			} else {
				unfinished = append(unfinished, v)
			}
		}
		if len(ready) != 0 {
			readyAggregation = append(readyAggregation, ready)
		}
		p.aggregation[k] = unfinished
	}
	p.aggMu.Unlock()

	// Finally send the events
	for _, v := range readyAggregation {
		p.flushCallback(p.cacheEntriesToLog(v))
	}
}

func (p *wafEventsPipeline) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// Flush the cache.
			p.Flush()
		case <-ctx.Done():
			return
		}
	}
}

func (p *wafEventsPipeline) aggregationAdd(entry *cacheEntry) {
	p.aggMu.Lock()
	defer p.aggMu.Unlock()

	key := aggregationKey{
		method:   entry.method,
		host:     entry.host,
		path:     entry.path,
		rulesKey: createRulesKey(entry),
	}
	entries, ok := p.aggregation[key]
	if !ok {
		entries = []*cacheEntry{}
	}
	entries = append(entries, entry)
	p.aggregation[key] = entries
}

func createRulesKey(entry *cacheEntry) uint64 {
	nonRulesHash := crc64.New(crc64.MakeTable(0))
	nonRulesHash.Write([]byte(entry.method))
	nonRulesHash.Write([]byte(entry.host))
	nonRulesHash.Write([]byte(entry.path))
	rulesHash := crc64.New(crc64.MakeTable(nonRulesHash.Sum64()))
	for _, rule := range entry.rules {
		rulesHash.Write([]byte(rule.Id))
	}
	return rulesHash.Sum64()
}
