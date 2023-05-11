// Copyright 2019 Tigera Inc. All rights reserved.

package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"

	lsv1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/projectcalico/calico/linseed/pkg/client"

	"github.com/araddon/dateparse"
	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"
	lmaAPI "github.com/projectcalico/calico/lma/pkg/api"
	lma "github.com/projectcalico/calico/lma/pkg/elastic"

	apiV3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

const (
	IPSetIndexPattern           = ".tigera.ipset.%s"
	DomainNameSetIndexPattern   = ".tigera.domainnameset.%s"
	ForwarderConfigIndexPattern = ".tigera.forwarderconfig.%s"
	WatchNamePrefixPattern      = "tigera_secure_ee_watch.%s."
	QuerySize                   = 1000
	MaxClauseCount              = 1024
	CreateIndexFailureDelay     = time.Second * 15
	CreateIndexWaitTimeout      = time.Minute
	DefaultReplicas             = 0
	DefaultShards               = 5
)

var (
	IPSetIndex           string
	DomainNameSetIndex   string
	WatchNamePrefix      string
	ForwarderConfigIndex string // ForwarderConfigIndex is an index for maintaining internal state for the event forwarder
)

func init() {
	clusterName := os.Getenv("CLUSTER_NAME")
	if clusterName == "" {
		clusterName = "cluster"
	}
	IPSetIndex = fmt.Sprintf(IPSetIndexPattern, clusterName)
	DomainNameSetIndex = fmt.Sprintf(DomainNameSetIndexPattern, clusterName)
	WatchNamePrefix = fmt.Sprintf(WatchNamePrefixPattern, clusterName)
	ForwarderConfigIndex = fmt.Sprintf(ForwarderConfigIndexPattern, clusterName)
}

type ipSetDoc struct {
	CreatedAt time.Time `json:"created_at"`
	IPs       IPSetSpec `json:"ips"`
}

type domainNameSetDoc struct {
	CreatedAt time.Time         `json:"created_at"`
	Domains   DomainNameSetSpec `json:"domains"`
}

type Service struct {
	lmaCLI                        lma.Client
	c                             *elastic.Client
	lsClient                      client.Client
	clusterName                   string
	ipSetMappingCreated           chan struct{}
	domainNameSetMappingCreated   chan struct{}
	eventMappingCreated           chan struct{}
	forwarderConfigMappingCreated chan struct{}
	cancel                        context.CancelFunc
	once                          sync.Once
	indexSettings                 IndexSettings
}

func NewService(lmaCLI lma.Client, lsClient client.Client, clusterName string, indexSettings IndexSettings) *Service {
	return &Service{
		lmaCLI:                        lmaCLI,
		c:                             lmaCLI.Backend(),
		lsClient:                      lsClient,
		clusterName:                   clusterName,
		ipSetMappingCreated:           make(chan struct{}),
		domainNameSetMappingCreated:   make(chan struct{}),
		eventMappingCreated:           make(chan struct{}),
		forwarderConfigMappingCreated: make(chan struct{}),
		indexSettings:                 indexSettings,
	}
}

func (e *Service) Run(ctx context.Context) {
	e.once.Do(func() {
		ctx, e.cancel = context.WithCancel(ctx)
		go func() {
			if err := CreateOrUpdateIndex(ctx, e.c, e.indexSettings, IPSetIndex, ipSetMapping, e.ipSetMappingCreated); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"index": IPSetIndex,
				}).Error("Could not create index")
			}
		}()

		go func() {
			if err := CreateOrUpdateIndex(ctx, e.c, e.indexSettings, DomainNameSetIndex, domainNameSetMapping, e.domainNameSetMappingCreated); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"index": DomainNameSetIndex,
				}).Error("Could not create index")
			}
		}()

		// We create the index ForwarderConfigIndex regardless of whether the event forwarder is enabled or not (so that it's
		// available when needed).
		go func() {
			if err := CreateOrUpdateIndex(ctx, e.c, e.indexSettings, ForwarderConfigIndex, forwarderConfigMapping, e.forwarderConfigMappingCreated); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"index": ForwarderConfigIndex,
				}).Error("Could not create index")
			}
		}()
	})
}

func (e *Service) Close() {
	e.cancel()
}

func (e *Service) ListIPSets(ctx context.Context) ([]Meta, error) {
	return e.listSets(ctx, IPSetIndex)
}

func (e *Service) ListDomainNameSets(ctx context.Context) ([]Meta, error) {
	return e.listSets(ctx, DomainNameSetIndex)
}

func (e *Service) listSets(ctx context.Context, idx string) ([]Meta, error) {
	q := elastic.NewMatchAllQuery()
	scroller := e.c.Scroll(idx).FetchSource(false).Query(q)
	defer scroller.Clear(ctx) // nolint: errcheck

	var ids []Meta
	for {
		res, err := scroller.Do(ctx)
		if err == io.EOF {
			return ids, nil
		}
		if elastic.IsNotFound(err) {
			// If we 404, just return an empty slice.
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		for _, hit := range res.Hits.Hits {
			ids = append(ids, Meta{
				Name:        hit.Id,
				SeqNo:       hit.SeqNo,
				PrimaryTerm: hit.PrimaryTerm,
			})
		}
	}
}

func (e *Service) PutIPSet(ctx context.Context, name string, set IPSetSpec) error {
	body := ipSetDoc{CreatedAt: time.Now(), IPs: set}
	return e.putSet(ctx, name, IPSetIndex, e.ipSetMappingCreated, body)
}

func (e *Service) PutDomainNameSet(ctx context.Context, name string, set DomainNameSetSpec) error {
	body := domainNameSetDoc{CreatedAt: time.Now(), Domains: set}
	return e.putSet(ctx, name, DomainNameSetIndex, e.domainNameSetMappingCreated, body)
}

func (e *Service) putSet(ctx context.Context, name string, idx string, c <-chan struct{}, body interface{}) error {
	// Wait for the Sets Mapping to be created
	if err := util.WaitForChannel(ctx, c, CreateIndexWaitTimeout); err != nil {
		return err
	}

	// Put document
	_, err := e.c.Index().Index(idx).Id(name).BodyJson(body).Do(ctx)
	log.WithField("name", name).Info("set stored")

	return err
}

func (e *Service) GetIPSet(ctx context.Context, name string) (IPSetSpec, error) {
	doc, err := e.get(ctx, IPSetIndex, name)
	if err != nil {
		return nil, err
	}
	i, ok := doc["ips"]
	if !ok {
		return nil, errors.New("document missing ips section")
	}

	ia, ok := i.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unknown type for %#v", i)
	}
	ips := IPSetSpec{}
	for _, v := range ia {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("unknown type for %#v", s)
		}
		ips = append(ips, s)
	}

	return ips, nil
}

func (e *Service) GetDomainNameSet(ctx context.Context, name string) (DomainNameSetSpec, error) {
	doc, err := e.get(ctx, DomainNameSetIndex, name)
	if err != nil {
		return nil, err
	}
	domains, ok := doc["domains"]
	if !ok {
		return nil, errors.New("document missing domains section")
	}

	idomains, ok := domains.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unknown type for %#v", domains)
	}
	result := DomainNameSetSpec{}
	for _, d := range idomains {
		s, ok := d.(string)
		if !ok {
			return nil, fmt.Errorf("unknown type for %#v", d)
		}
		result = append(result, s)
	}

	return result, nil
}

func (e *Service) get(ctx context.Context, idx, name string) (map[string]interface{}, error) {
	res, err := e.c.Get().Index(idx).Id(name).Do(ctx)
	if err != nil {
		return nil, err
	}

	if res.Source == nil {
		return nil, errors.New("Elastic document has nil Source")
	}

	var doc map[string]interface{}
	err = json.Unmarshal(res.Source, &doc)
	return doc, err
}

func (e *Service) GetIPSetModified(ctx context.Context, name string) (time.Time, error) {
	return e.getSetModified(ctx, name, IPSetIndex)
}

func (e *Service) GetDomainNameSetModified(ctx context.Context, name string) (time.Time, error) {
	return e.getSetModified(ctx, name, DomainNameSetIndex)
}

func (e *Service) getSetModified(ctx context.Context, name, idx string) (time.Time, error) {
	res, err := e.c.Get().Index(idx).Id(name).FetchSourceContext(elastic.NewFetchSourceContext(true).Include("created_at")).Do(ctx)
	if err != nil {
		return time.Time{}, err
	}
	if res.Source == nil {
		return time.Time{}, err
	}
	var doc map[string]interface{}
	err = json.Unmarshal(res.Source, &doc)
	if err != nil {
		return time.Time{}, err
	}
	createdAt, ok := doc["created_at"]
	if !ok {
		// missing created_at field
		return time.Time{}, nil
	}
	switch createdAt := createdAt.(type) {
	case string:
		return dateparse.ParseIn(createdAt, time.UTC)
	default:
		return time.Time{}, fmt.Errorf("unexpected type for %#v", createdAt)
	}
}

type SetQuerier interface {
	// QueryIPSet queries the flow log by IPs specified in the feed's IPSet.
	// It returns a queryIterator, the latest IPSet hash, and error if any happens during the querying
	QueryIPSet(ctx context.Context, feed *apiV3.GlobalThreatFeed) (queryIterator Iterator[lsv1.FlowLog], newSetHash string, err error)
	// QueryDomainNameSet queries the DNS log by domain names specified in the feed's DomainNameSet.
	// It returns a queryIterator, the latest DomainNameSet hash, and error if any happens during the querying
	QueryDomainNameSet(ctx context.Context, set DomainNameSetSpec, feed *apiV3.GlobalThreatFeed) (queryIterator Iterator[lsv1.DNSLog], newSetHash string, err error)
	// GetDomainNameSet queries and outputs all the domain names specified in the feed's DomainNameSet.
	GetDomainNameSet(ctx context.Context, name string) (DomainNameSetSpec, error)
}

func (e *Service) QueryIPSet(ctx context.Context, feed *apiV3.GlobalThreatFeed) (Iterator[lsv1.FlowLog], string, error) {
	ipset, err := e.GetIPSet(ctx, feed.Name)
	if err != nil {
		return nil, "", err
	}

	newIpSetHash := util.ComputeSha256Hash(ipset)
	var fromTimestamp time.Time
	currentIpSetHash := feed.Annotations[IpSetHashKey]
	// If the ipSet has changed we need to query from the beginning of time, otherwise query from the last successful time
	if feed.Status.LastSuccessfulSearch != nil && strings.Compare(newIpSetHash, currentIpSetHash) == 0 {
		fromTimestamp = feed.Status.LastSuccessfulSearch.Time
	}

	// Create the list pager for flow logs
	var tr lmav1.TimeRange
	tr.From = fromTimestamp
	tr.To = time.Now()

	queryTerms := splitIPSet(ipset)
	var queries []queryEntry[lsv1.FlowLog, lsv1.FlowLogParams]

	for _, t := range queryTerms {
		matchSource := flowParams(tr, lsv1.MatchTypeSource, t)
		queries = append(queries, queryEntry[lsv1.FlowLog, lsv1.FlowLogParams]{
			key:         QueryKeyFlowLogSourceIP,
			queryParams: matchSource,
			listPager:   client.NewListPager[lsv1.FlowLog](&matchSource),
			listFn:      e.lsClient.FlowLogs(e.clusterName).List,
		})

		matchDestination := flowParams(tr, lsv1.MatchTypeDest, t)
		queries = append(queries, queryEntry[lsv1.FlowLog, lsv1.FlowLogParams]{
			key:         QueryKeyFlowLogDestIP,
			queryParams: matchDestination,
			listPager:   client.NewListPager[lsv1.FlowLog](&matchDestination),
			listFn:      e.lsClient.FlowLogs(e.clusterName).List,
		})
	}

	return newQueryIterator(ctx, queries, feed.Name), newIpSetHash, nil
}

func flowParams(tr lmav1.TimeRange, matchType lsv1.MatchType, t []string) lsv1.FlowLogParams {
	matchSource := lsv1.FlowLogParams{QueryParams: lsv1.QueryParams{TimeRange: &tr}}
	matchSource.IPMatches = []lsv1.IPMatch{
		{
			Type: matchType,
			IPs:  t,
		},
	}
	matchSource.SetMaxPageSize(QuerySize)
	return matchSource
}

func (e *Service) QueryDomainNameSet(ctx context.Context, domainNameSet DomainNameSetSpec, feed *apiV3.GlobalThreatFeed) (Iterator[lsv1.DNSLog], string, error) {
	newDomainNameSetHash := util.ComputeSha256Hash(domainNameSet)
	var fromTimestamp time.Time
	currentDomainNameSetHash := feed.Annotations[DomainNameSetHashKey]
	// If the domainNameSet has changed we need to query from the beginning of time, otherwise query from the last successful time
	if feed.Status.LastSuccessfulSearch != nil && strings.Compare(newDomainNameSetHash, currentDomainNameSetHash) == 0 {
		fromTimestamp = feed.Status.LastSuccessfulSearch.Time
	}

	queryTerms := splitDomainNameSet(domainNameSet)

	// Ordering is important for the queries, so that we get more relevant results earlier. The caller
	// wants to de-duplicate events that point to the same DNS query. For example, a DNS query for www.example.com
	// will create a DNS log with www.example.com in both the qname and one of the rrsets.name. We only want to emit
	// one security event in this case, and the most relevant one is the one that says a pod queried directly for
	// for a name on our threat list.

	// Create the list pager for flow logs
	var tr lmav1.TimeRange
	tr.From = fromTimestamp
	tr.To = time.Now()

	var queries []queryEntry[lsv1.DNSLog, lsv1.DNSLogParams]
	for _, t := range queryTerms {
		matchQname := dnsLogParams(tr, lsv1.DomainMatchQname, t)
		queries = append(queries, queryEntry[lsv1.DNSLog, lsv1.DNSLogParams]{
			key:         QueryKeyDNSLogQName,
			queryParams: matchQname,
			listPager:   client.NewListPager[lsv1.DNSLog](&matchQname), listFn: e.lsClient.DNSLogs(e.clusterName).List,
		})
		matchRRSet := dnsLogParams(tr, lsv1.DomainMatchRRSet, t)
		queries = append(queries, queryEntry[lsv1.DNSLog, lsv1.DNSLogParams]{
			key:       QueryKeyDNSLogRRSetsName,
			listPager: client.NewListPager[lsv1.DNSLog](&matchRRSet), listFn: e.lsClient.DNSLogs(e.clusterName).List,
		})
		matchRRData := dnsLogParams(tr, lsv1.DomainMatchRRData, t)
		queries = append(queries, queryEntry[lsv1.DNSLog, lsv1.DNSLogParams]{
			key:       QueryKeyDNSLogRRSetsRData,
			listPager: client.NewListPager[lsv1.DNSLog](&matchRRData), listFn: e.lsClient.DNSLogs(e.clusterName).List,
		})
	}

	return newQueryIterator(ctx, queries, feed.Name), newDomainNameSetHash, nil
}

func dnsLogParams(tr lmav1.TimeRange, matchType lsv1.DomainMatchType, domainNameSet DomainNameSetSpec) lsv1.DNSLogParams {
	matchQname := lsv1.DNSLogParams{QueryParams: lsv1.QueryParams{TimeRange: &tr}}
	matchQname.DomainMatches = []lsv1.DomainMatch{
		{
			Type:    matchType,
			Domains: domainNameSet,
		},
	}
	matchQname.SetMaxPageSize(QuerySize)
	return matchQname
}

func splitIPSet(ipset IPSetSpec) [][]string {
	return splitStringSlice(ipset)
}

func splitDomainNameSet(set DomainNameSetSpec) [][]string {
	return splitStringSlice(set)
}

func splitStringSlice(set []string) [][]string {
	terms := make([][]string, 1)
	for _, t := range set {
		if len(terms[len(terms)-1]) >= MaxClauseCount {
			terms = append(terms, []string{t})
		} else {
			terms[len(terms)-1] = append(terms[len(terms)-1], t)
		}
	}
	return terms
}

func (e *Service) DeleteIPSet(ctx context.Context, m Meta) error {
	return e.deleteSet(ctx, m, IPSetIndex)
}

func (e *Service) DeleteDomainNameSet(ctx context.Context, m Meta) error {
	return e.deleteSet(ctx, m, DomainNameSetIndex)
}

func (e *Service) deleteSet(ctx context.Context, m Meta, idx string) error {
	ds := e.c.Delete().Index(idx).Id(m.Name)
	if m.SeqNo != nil {
		ds = ds.IfSeqNo(*m.SeqNo)
	}
	if m.PrimaryTerm != nil {
		ds = ds.IfPrimaryTerm(*m.PrimaryTerm)
	}
	_, err := ds.Do(ctx)
	return err
}

func (e *Service) PutSecurityEventWithID(ctx context.Context, f []lsv1.Event) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if len(f) == 0 {
		return nil
	}
	_, err := e.lsClient.Events(e.clusterName).Create(ctx, f)
	return err
}

// GetSecurityEvents retrieves a listing of security events from ES sorted in ascending order,
// where each events time falls within the range given by start and end time.
func (e *Service) GetSecurityEvents(ctx context.Context, start, end time.Time, allClusters bool) <-chan *lmaAPI.EventResult {
	return e.lmaCLI.SearchSecurityEvents(ctx, &start, &end, nil, allClusters)
}

// PutForwarderConfig saves the given ForwarderConfig object back to the datastore.
func (e *Service) PutForwarderConfig(ctx context.Context, id string, f *ForwarderConfig) error {
	l := log.WithFields(log.Fields{"func": "PutForwarderConfig"})
	// Wait for the SecurityEvent Mapping to be created
	if err := util.WaitForChannel(ctx, e.forwarderConfigMappingCreated, CreateIndexWaitTimeout); err != nil {
		return err
	}
	l.Debugf("Save config with id[%s] content [%+v]", id, f)
	resp, err := e.c.Index().Index(ForwarderConfigIndex).Id(id).BodyJson(f).Do(ctx)
	l.Debugf("Save config response [%+v]", resp)
	return err
}

// GetForwarderConfig retrieves the forwarder config (which will be a singleton).
func (e *Service) GetForwarderConfig(ctx context.Context, id string) (*ForwarderConfig, error) {
	l := log.WithFields(log.Fields{"func": "GetForwarderConfig"})
	l.Debugf("Search for config with id[%s]", id)

	searchResult, err := e.c.Search().
		Index(ForwarderConfigIndex).
		Query(elastic.NewTermQuery("_id", id)). // query for the singleton doc by ID
		From(0).Size(1).                        // retrieve the single document containing the forwarder config
		Do(ctx)                                 // execute
	if err != nil {
		return nil, err
	}

	l.Debugf("Query took %d milliseconds", searchResult.TookInMillis)
	l.Debugf("Found a total of %d forwarder config", searchResult.TotalHits())

	if searchResult.Hits.TotalHits.Value > 0 {
		hit := searchResult.Hits.Hits[0]
		l.Debugf("Selecting forwarder config with id[%s]", hit.Id)
		var config ForwarderConfig
		if err := json.Unmarshal(hit.Source, &config); err != nil {
			return nil, err
		}
		return &config, nil
	}

	return nil, nil
}
