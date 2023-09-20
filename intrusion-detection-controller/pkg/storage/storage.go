// Copyright 2019 Tigera Inc. All rights reserved.

package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/intrusion-detection-controller/pkg/util"
	lsv1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
	lmaAPI "github.com/projectcalico/calico/lma/pkg/api"
	lmav1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	lma "github.com/projectcalico/calico/lma/pkg/elastic"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"

	apiV3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

const (
	ForwarderConfigIndexPattern = ".tigera.forwarderconfig.%s"
	QuerySize                   = 1000
	MaxClauseCount              = 1024
	CreateIndexFailureDelay     = time.Second * 15
	CreateIndexWaitTimeout      = time.Minute
	DefaultReplicas             = 0
	DefaultShards               = 5
)

var (
	ForwarderConfigIndex string // ForwarderConfigIndex is an index for maintaining internal state for the event forwarder
)

func init() {
	clusterName := os.Getenv("ELASTIC_INDEX_SUFFIX")
	if clusterName == "" {
		clusterName = lmak8s.DefaultCluster
	}
	ForwarderConfigIndex = fmt.Sprintf(ForwarderConfigIndexPattern, clusterName)
}

type Service struct {
	lmaCLI                        lma.Client
	c                             *elastic.Client
	lsClient                      client.Client
	clusterName                   string
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
		forwarderConfigMappingCreated: make(chan struct{}),
		indexSettings:                 indexSettings,
	}
}

func (e *Service) Run(ctx context.Context) {
	e.once.Do(func() {
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
	if e.cancel != nil {
		e.cancel()
	}
}

func (e *Service) ListIPSets(ctx context.Context) ([]Meta, error) {
	pager := client.NewListPager[lsv1.IPSetThreatFeed](&lsv1.IPSetThreatFeedParams{})
	pages, errors := pager.Stream(ctx, e.lsClient.ThreatFeeds(e.clusterName).IPSet().List)

	var ids []Meta
	for page := range pages {
		for _, item := range page.Items {
			ids = append(ids, Meta{
				Name:        item.ID,
				SeqNo:       item.SeqNumber,
				PrimaryTerm: item.PrimaryTerm,
			})
		}
	}

	if err, ok := <-errors; ok {
		log.WithError(err).Error("failed to read thread feeds")
		return nil, err
	}

	return ids, nil
}

func (e *Service) ListDomainNameSets(ctx context.Context) ([]Meta, error) {
	pager := client.NewListPager[lsv1.DomainNameSetThreatFeed](&lsv1.DomainNameSetThreatFeedParams{})
	pages, errors := pager.Stream(ctx, e.lsClient.ThreatFeeds(e.clusterName).DomainNameSet().List)

	var ids []Meta
	for page := range pages {
		for _, item := range page.Items {
			ids = append(ids, Meta{
				Name:        item.ID,
				SeqNo:       item.SeqNumber,
				PrimaryTerm: item.PrimaryTerm,
			})
		}
	}

	if err, ok := <-errors; ok {
		log.WithError(err).Error("failed to read thread feeds")
		return nil, err
	}

	return ids, nil
}

func (e *Service) PutIPSet(ctx context.Context, name string, set IPSetSpec) error {
	feed := lsv1.IPSetThreatFeed{
		ID: name,
		Data: &lsv1.IPSetThreatFeedData{
			CreatedAt: time.Now().UTC(),
			IPs:       set,
		},
	}

	response, err := e.lsClient.ThreatFeeds(e.clusterName).IPSet().Create(ctx, []lsv1.IPSetThreatFeed{feed})
	bulkErr := e.checkBulkError(err, response)
	if bulkErr != nil {
		return bulkErr
	}

	return nil
}

func (e *Service) checkBulkError(err error, response *lsv1.BulkResponse) error {
	if err != nil {
		return err
	}
	if response.Failed != 0 {
		var errorMsg []string
		for _, msg := range response.Errors {
			errorMsg = append(errorMsg, msg.Error())
		}

		return fmt.Errorf(strings.Join(errorMsg, " and "))
	}
	return nil
}

func (e *Service) PutDomainNameSet(ctx context.Context, name string, set DomainNameSetSpec) error {
	feed := lsv1.DomainNameSetThreatFeed{
		ID: name,
		Data: &lsv1.DomainNameSetThreatFeedData{
			CreatedAt: time.Now(),
			Domains:   set,
		},
	}

	response, err := e.lsClient.ThreatFeeds(e.clusterName).DomainNameSet().Create(ctx, []lsv1.DomainNameSetThreatFeed{feed})
	bulkErr := e.checkBulkError(err, response)
	if bulkErr != nil {
		return bulkErr
	}

	return nil
}

func (e *Service) GetIPSet(ctx context.Context, name string) (IPSetSpec, error) {
	params := lsv1.IPSetThreatFeedParams{
		ID: name,
	}

	response, err := e.lsClient.ThreatFeeds(e.clusterName).IPSet().List(ctx, &params)
	if err != nil {
		return nil, err
	}

	var data []string
	for _, item := range response.Items {
		if item.Data != nil {
			data = append(data, item.Data.IPs...)
		}
	}

	return data, nil
}

func (e *Service) GetDomainNameSet(ctx context.Context, name string) (DomainNameSetSpec, error) {
	params := lsv1.DomainNameSetThreatFeedParams{
		ID: name,
	}

	response, err := e.lsClient.ThreatFeeds(e.clusterName).DomainNameSet().List(ctx, &params)
	if err != nil {
		return nil, err
	}

	var data []string
	for _, item := range response.Items {
		if item.Data != nil {
			data = append(data, item.Data.Domains...)
		}
	}

	return data, nil
}

func (e *Service) GetIPSetModified(ctx context.Context, name string) (time.Time, error) {
	params := lsv1.IPSetThreatFeedParams{
		ID: name,
	}
	response, err := e.lsClient.ThreatFeeds(e.clusterName).IPSet().List(ctx, &params)
	if err != nil {
		return time.Time{}, err
	}

	if response.TotalHits != 1 {
		return time.Time{}, fmt.Errorf("multiple feeds returned for name")
	}

	if response.Items[0].Data != nil {
		createdAt := response.Items[0].Data.CreatedAt
		return createdAt, nil
	}

	return time.Time{}, fmt.Errorf("missing created time field")
}

func (e *Service) GetDomainNameSetModified(ctx context.Context, name string) (time.Time, error) {
	params := lsv1.DomainNameSetThreatFeedParams{
		ID: name,
	}

	response, err := e.lsClient.ThreatFeeds(e.clusterName).DomainNameSet().List(ctx, &params)
	if err != nil {
		return time.Time{}, err
	}

	if response.TotalHits != 1 {
		return time.Time{}, fmt.Errorf("multiple feeds returned for name")
	}

	if response.Items[0].Data != nil {
		createdAt := response.Items[0].Data.CreatedAt
		return createdAt, nil
	}

	return time.Time{}, fmt.Errorf("missing created time field")
}

func (e *Service) DeleteIPSet(ctx context.Context, m Meta) error {
	feed := lsv1.IPSetThreatFeed{
		ID:          m.Name,
		SeqNumber:   m.SeqNo,
		PrimaryTerm: m.PrimaryTerm,
	}

	response, err := e.lsClient.ThreatFeeds(e.clusterName).IPSet().Delete(ctx, []lsv1.IPSetThreatFeed{feed})
	bulkErr := e.checkBulkError(err, response)
	if bulkErr != nil {
		return bulkErr
	}

	return nil
}

func (e *Service) DeleteDomainNameSet(ctx context.Context, m Meta) error {
	feed := lsv1.DomainNameSetThreatFeed{
		ID:          m.Name,
		SeqNumber:   m.SeqNo,
		PrimaryTerm: m.PrimaryTerm,
	}

	response, err := e.lsClient.ThreatFeeds(e.clusterName).DomainNameSet().Delete(ctx, []lsv1.DomainNameSetThreatFeed{feed})
	bulkErr := e.checkBulkError(err, response)
	if bulkErr != nil {
		return bulkErr
	}

	return nil
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
