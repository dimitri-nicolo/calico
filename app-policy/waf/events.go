package waf

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	corazatypes "github.com/corazawaf/coraza/v3/types"

	"github.com/projectcalico/calico/app-policy/checker"
	"github.com/projectcalico/calico/app-policy/policystore"
	linseedv1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

type cacheKey struct {
	destIP string
}

type cacheEntry struct {
	transactionID         string
	destIP, srcIP         string
	method, protocol      string
	srcPort, dstPort      uint32
	uri                   map[string]int
	rules                 map[int]linseedv1.WAFRuleHit
	count                 int
	action                string
	srcName, srcNamespace string
}

type wafEventsPipeline struct {
	store         policystore.PolicyStoreManager
	cache         map[cacheKey]*cacheEntry
	flushCallback eventCallbackFn
	mu            sync.Locker
}

type txHttpInfo struct {
	txID, destIP          string
	uri, method, protocol string
	srcPort, dstPort      uint32
	action                string
	srcName, srcNamespace string
}

func NewEventsPipeline(store policystore.PolicyStoreManager, cb eventCallbackFn) *wafEventsPipeline {
	return &wafEventsPipeline{
		store:         store,
		cache:         make(map[cacheKey]*cacheEntry),
		flushCallback: cb,
		mu:            &sync.Mutex{},
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
func (p *wafEventsPipeline) Process(event interface{}) {
	switch e := event.(type) {
	case corazatypes.MatchedRule:
		p.processMatchedRule(e)
	case *txHttpInfo:
		p.processTxHttpInfo(e)
	default:
		log.WithField("event", e).Warn("Unknown event type")
	}
}

func (p *wafEventsPipeline) processMatchedRule(matchedRule corazatypes.MatchedRule) {
	rule := matchedRule.Rule()
	if rule == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	log.WithFields(log.Fields{
		"rule":  rule.ID(),
		"data":  matchedRule.Data(),
		"audit": matchedRule.AuditLog(),
	}).Debug("Processing matched rule")

	key := cacheKey{matchedRule.ServerIPAddress()}
	entry, ok := p.cache[key]
	if !ok {
		entry = &cacheEntry{
			transactionID: matchedRule.TransactionID(),
			destIP:        matchedRule.ServerIPAddress(),
			srcIP:         matchedRule.ClientIPAddress(),
			uri:           make(map[string]int),
			rules:         make(map[int]linseedv1.WAFRuleHit),
		}
		p.cache[key] = entry
	}

	entry.count++
	entry.uri[matchedRule.URI()]++
	entry.rules = mergeHits(entry.rules, corazaRulesToLinseedWAFRuleHit(matchedRule))

}

func (p *wafEventsPipeline) processTxHttpInfo(info *txHttpInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := cacheKey{info.destIP}
	entry, ok := p.cache[key]
	if !ok {
		// only update the cache if we have a matching entry
		return
	}

	// fill in missing info
	entry.method = info.method
	entry.protocol = info.protocol
	entry.srcPort = info.srcPort
	entry.dstPort = info.dstPort
	entry.action = info.action
	entry.srcName = info.srcName
	entry.srcNamespace = info.srcNamespace
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func (p *wafEventsPipeline) cacheEntryToLog(entry cacheEntry) *linseedv1.WAFLog {
	var srcEp, dstEp *linseedv1.WAFEndpoint
	p.store.Read(
		func(ps *policystore.PolicyStore) {
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

			dstNamespace, dstName := extractFirstWepNameAndNamespace(dst)
			srcNamespaceFromCache, srcNameFromCache := extractFirstWepNameAndNamespace(src)

			// prioritize the info from the cache
			srcName := firstNonEmpty(srcNameFromCache, entry.srcName)
			srcNamespace := firstNonEmpty(srcNamespaceFromCache, entry.srcNamespace)

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
		},
	)

	log := &linseedv1.WAFLog{
		RequestId:   entry.transactionID,
		Source:      srcEp,
		Destination: dstEp,
		Rules:       unMapHits(entry.rules),
		Msg:         fmt.Sprintf("WAF detected %d violations %s", entry.count, bracket(entry.action)),
		// path needs to be aggregated in the future
		Path: mostFrequentURI(entry.uri),
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

func (p *wafEventsPipeline) Flush() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for k, v := range p.cache {
		p.flushCallback(p.cacheEntryToLog(*v))
		delete(p.cache, k)
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

func corazaRulesToLinseedWAFRuleHit(matchedRule corazatypes.MatchedRule) (hits map[int]linseedv1.WAFRuleHit) {
	ruleInfo := matchedRule.Rule()
	return map[int]linseedv1.WAFRuleHit{
		ruleInfo.ID(): {
			Message:    matchedRule.Message(),
			Disruptive: matchedRule.Disruptive(),
			Id:         fmt.Sprint(ruleInfo.ID()),
			Severity:   ruleInfo.Severity().String(),
			File:       ruleInfo.File(),
			Line:       fmt.Sprint(ruleInfo.Line()),
		},
	}
}

func mergeHits(a, b map[int]linseedv1.WAFRuleHit) map[int]linseedv1.WAFRuleHit {
	for k, v := range b {
		a[k] = v
	}
	return a
}

func unMapHits(hits map[int]linseedv1.WAFRuleHit) (hitList []linseedv1.WAFRuleHit) {
	for _, v := range hits {
		hitList = append(hitList, v)
	}
	return hitList
}

func mostFrequentURI(uri map[string]int) string {
	var max int
	var maxURI string
	for k, v := range uri {
		if v > max {
			max = v
			maxURI = k
		}
	}
	return maxURI
}
