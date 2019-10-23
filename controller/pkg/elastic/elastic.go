// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/araddon/dateparse"
	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
)

const (
	IPSetIndexPattern         = ".tigera.ipset.%s"
	DomainNameSetIndexPattern = ".tigera.domainnameset.%s"
	FlowLogIndexPattern       = "tigera_secure_ee_flows.%s.*"
	DNSLogIndexPattern        = "tigera_secure_ee_dns.%s.*"
	EventIndexPattern         = "tigera_secure_ee_events.%s"
	AuditIndexPattern         = "tigera_secure_ee_audit_*.%s.*"
	WatchIndex                = ".watches"
	WatchNamePrefixPattern    = "tigera_secure_ee_watch.%s."
	QuerySize                 = 1000
	AuditQuerySize            = 0
	MaxClauseCount            = 1024
	CreateIndexFailureDelay   = time.Second * 15
	CreateIndexWaitTimeout    = time.Minute
	PingTimeout               = time.Second * 5
	PingPeriod                = time.Minute
	Create                    = "create"
	Delete                    = "delete"
)

var (
	IPSetIndex         string
	DomainNameSetIndex string
	EventIndex         string
	FlowLogIndex       string
	DNSLogIndex        string
	AuditIndex         string
	WatchNamePrefix    string
)

func init() {
	clusterName := os.Getenv("CLUSTER_NAME")
	if clusterName == "" {
		clusterName = "cluster"
	}
	IPSetIndex = fmt.Sprintf(IPSetIndexPattern, clusterName)
	DomainNameSetIndex = fmt.Sprintf(DomainNameSetIndexPattern, clusterName)
	EventIndex = fmt.Sprintf(EventIndexPattern, clusterName)
	FlowLogIndex = fmt.Sprintf(FlowLogIndexPattern, clusterName)
	DNSLogIndex = fmt.Sprintf(DNSLogIndexPattern, clusterName)
	AuditIndex = fmt.Sprintf(AuditIndexPattern, clusterName)
	WatchNamePrefix = fmt.Sprintf(WatchNamePrefixPattern, clusterName)
}

type ipSetDoc struct {
	CreatedAt time.Time    `json:"created_at"`
	IPs       db.IPSetSpec `json:"ips"`
}

type domainNameSetDoc struct {
	CreatedAt time.Time            `json:"created_at"`
	Domains   db.DomainNameSetSpec `json:"domains"`
}

type Elastic struct {
	c                           *elastic.Client
	url                         *url.URL
	ipSetMappingCreated         chan struct{}
	domainNameSetMappingCreated chan struct{}
	eventMappingCreated         chan struct{}
	elasticIsAlive              bool
	cancel                      context.CancelFunc
	once                        sync.Once
}

func NewElastic(h *http.Client, url *url.URL, username, password string) (*Elastic, error) {

	options := []elastic.ClientOptionFunc{
		elastic.SetURL(url.String()),
		elastic.SetHttpClient(h),
		elastic.SetErrorLog(log.StandardLogger()),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
		//elastic.SetTraceLog(log.StandardLogger()),
	}
	if username != "" {
		options = append(options, elastic.SetBasicAuth(username, password))
	}
	c, err := elastic.NewClient(options...)
	if err != nil {
		return nil, err
	}
	e := &Elastic{
		c:                           c,
		url:                         url,
		ipSetMappingCreated:         make(chan struct{}),
		domainNameSetMappingCreated: make(chan struct{}),
		eventMappingCreated:         make(chan struct{}),
	}

	return e, nil
}

func (e *Elastic) Run(ctx context.Context) {
	e.once.Do(func() {
		ctx, e.cancel = context.WithCancel(ctx)
		go func() {
			if err := e.createOrUpdateIndex(ctx, IPSetIndex, ipSetMapping, e.ipSetMappingCreated); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"index": IPSetIndex,
				}).Error("Could not create index")
			}
		}()
		go func() {
			if err := e.createOrUpdateIndex(ctx, DomainNameSetIndex, domainNameSetMapping, e.domainNameSetMappingCreated); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"index": DomainNameSetIndex,
				}).Error("Could not create index")
			}
		}()

		go func() {
			if err := e.createOrUpdateIndex(ctx, EventIndex, EventMapping, e.eventMappingCreated); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"index": EventIndex,
				}).Error("Could not create index")
			}
		}()

		go func() {
			if err := runloop.RunLoop(ctx, func() {
				childCtx, cancel := context.WithTimeout(ctx, PingTimeout)
				defer cancel()

				_, code, err := e.c.Ping(e.url.String()).Do(childCtx)
				switch {
				case err != nil:
					log.WithError(err).Warn("Elastic ping failed")
					e.elasticIsAlive = false
				case code < http.StatusOK || code >= http.StatusBadRequest:
					log.WithField("code", code).Warn("Elastic ping failed")
					e.elasticIsAlive = false
				default:
					e.elasticIsAlive = true
				}
			}, PingPeriod); err != nil {
				log.WithError(err).Error("Elastic ping failed")
			}
		}()
	})
}

func (e *Elastic) Close() {
	e.cancel()
}

func (e *Elastic) ListIPSets(ctx context.Context) ([]db.Meta, error) {
	return e.listSets(ctx, IPSetIndex)
}

func (e *Elastic) ListDomainNameSets(ctx context.Context) ([]db.Meta, error) {
	return e.listSets(ctx, DomainNameSetIndex)
}

func (e *Elastic) listSets(ctx context.Context, idx string) ([]db.Meta, error) {
	q := elastic.NewMatchAllQuery()
	scroller := e.c.Scroll(idx).Version(true).FetchSource(false).Query(q)
	defer scroller.Clear(ctx)

	var ids []db.Meta
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
			ids = append(ids, db.Meta{Name: hit.Id, Version: hit.Version})
		}
	}
}

func (e *Elastic) PutIPSet(ctx context.Context, name string, set db.IPSetSpec) error {
	body := ipSetDoc{CreatedAt: time.Now(), IPs: set}
	return e.putSet(ctx, name, IPSetIndex, e.ipSetMappingCreated, body)
}

func (e *Elastic) PutDomainNameSet(ctx context.Context, name string, set db.DomainNameSetSpec) error {
	body := domainNameSetDoc{CreatedAt: time.Now(), Domains: set}
	return e.putSet(ctx, name, DomainNameSetIndex, e.domainNameSetMappingCreated, body)
}

func (e *Elastic) putSet(ctx context.Context, name string, idx string, c <-chan struct{}, body interface{}) error {

	// Wait for the Sets Mapping to be created
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(CreateIndexWaitTimeout):
		return errors.New("Timeout waiting for index creation")
	case <-c:
		break
	}

	// Put document
	_, err := e.c.Index().Index(idx).Id(name).BodyJson(body).Do(ctx)
	log.WithField("name", name).Info("set stored")

	return err
}

func (e *Elastic) createOrUpdateIndex(ctx context.Context, index, mapping string, ch chan struct{}) error {
	attempt := 0
	for {
		attempt++
		err := e.ensureIndexExists(ctx, index, mapping)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"index":       index,
				"attempt":     attempt,
				"retry_delay": CreateIndexFailureDelay,
			}).Errorf("Failed to create index")

			select {
			case <-ctx.Done():
				return err
			case <-time.After(CreateIndexFailureDelay):
				// noop
			}
		} else {
			close(ch)
			return nil
		}
	}
}

func (e *Elastic) ensureIndexExists(ctx context.Context, idx, mapping string) error {
	exists, err := e.c.IndexExists(idx).Do(ctx)
	if err != nil {
		return err
	}
	if !exists {
		r, err := e.c.CreateIndex(idx).Body(mapping).Do(ctx)
		if err != nil {
			return err
		}
		if !r.Acknowledged {
			return fmt.Errorf("not acknowledged index %s create", idx)
		}
	}
	return nil
}

func (e *Elastic) GetIPSet(ctx context.Context, name string) (db.IPSetSpec, error) {
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
	ips := db.IPSetSpec{}
	for _, v := range ia {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("unknown type for %#v", s)
		}
		ips = append(ips, s)
	}

	return ips, nil
}

func (e *Elastic) GetDomainNameSet(ctx context.Context, name string) (db.DomainNameSetSpec, error) {
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
	result := db.DomainNameSetSpec{}
	for _, d := range idomains {
		s, ok := d.(string)
		if !ok {
			return nil, fmt.Errorf("unknown type for %#v", d)
		}
		result = append(result, s)
	}

	return result, nil
}

func (e *Elastic) get(ctx context.Context, idx, name string) (map[string]interface{}, error) {
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

func (e *Elastic) GetIPSetModified(ctx context.Context, name string) (time.Time, error) {
	return e.getSetModified(ctx, name, IPSetIndex)
}

func (e *Elastic) GetDomainNameSetModified(ctx context.Context, name string) (time.Time, error) {
	return e.getSetModified(ctx, name, DomainNameSetIndex)
}

func (e *Elastic) getSetModified(ctx context.Context, name, idx string) (time.Time, error) {
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
	switch createdAt.(type) {
	case string:
		return dateparse.ParseIn(createdAt.(string), time.UTC)
	default:
		return time.Time{}, fmt.Errorf("Unexpected type for %#v", createdAt)
	}
}

type SetQuerier interface {
	QueryIPSet(ctx context.Context, name string) (Iterator, error)
	QueryDomainNameSet(ctx context.Context, name string, set db.DomainNameSetSpec) (Iterator, error)
	GetDomainNameSet(ctx context.Context, name string) (db.DomainNameSetSpec, error)
}

func (e *Elastic) QueryIPSet(ctx context.Context, name string) (Iterator, error) {
	ipset, err := e.GetIPSet(ctx, name)
	if err != nil {
		return nil, err
	}
	queryTerms := splitIPSetToInterface(ipset)

	f := func(ipset, field string, terms []interface{}) *elastic.ScrollService {
		q := elastic.NewTermsQuery(field, terms...)
		return e.c.Scroll(FlowLogIndex).SortBy(elastic.SortByDoc{}).Query(q).Size(QuerySize)
	}

	var scrollers []scrollerEntry
	for _, t := range queryTerms {
		scrollers = append(scrollers, scrollerEntry{key: db.QueryKeyFlowLogSourceIP, scroller: f(name, "source_ip", t), terms: t})
		scrollers = append(scrollers, scrollerEntry{key: db.QueryKeyFlowLogDestIP, scroller: f(name, "dest_ip", t), terms: t})
	}

	return newQueryIterator(ctx, scrollers, name), nil
}

func (e *Elastic) QueryDomainNameSet(ctx context.Context, name string, set db.DomainNameSetSpec) (Iterator, error) {
	queryTerms := splitDomainNameSetToInterface(set)

	var scrollers []scrollerEntry

	// Ordering is important for the scrollers, so that we get more relevant results earlier. The caller
	// wants to de-duplicate events that point to the same DNS query. For example, a DNS query for www.example.com
	// will create a DNS log with www.example.com in both the qname and one of the rrsets.name. We only want to emit
	// one security event in this case, and the most relevant one is the one that says a pod queried directly for
	// for a name on our threat list.

	// QName scrollers
	for _, t := range queryTerms {
		qname := e.c.Scroll(DNSLogIndex).
			SortBy(elastic.SortByDoc{}).
			Size(QuerySize).
			Query(elastic.NewTermsQuery("qname", t...))
		scrollers = append(scrollers, scrollerEntry{key: db.QueryKeyDNSLogQName, scroller: qname, terms: t})
	}

	// RRSet.name scrollers
	for _, t := range queryTerms {
		rrsn := e.c.Scroll(DNSLogIndex).
			SortBy(elastic.SortByDoc{}).
			Size(QuerySize).
			Query(
				elastic.NewNestedQuery(
					"rrsets",
					elastic.NewTermsQuery("rrsets.name", t...),
				),
			)
		scrollers = append(scrollers, scrollerEntry{key: db.QueryKeyDNSLogRRSetsName, scroller: rrsn, terms: t})
	}

	// RRSet.rdata scrollers
	for _, t := range queryTerms {
		rrsrd := e.c.Scroll(DNSLogIndex).
			SortBy(elastic.SortByDoc{}).
			Size(QuerySize).
			Query(
				elastic.NewNestedQuery(
					"rrsets",
					elastic.NewTermsQuery("rrsets.rdata", t...),
				),
			)
		scrollers = append(scrollers, scrollerEntry{key: db.QueryKeyDNSLogRRSetsRData, scroller: rrsrd, terms: t})
	}
	return newQueryIterator(ctx, scrollers, name), nil
}

func splitIPSetToInterface(ipset db.IPSetSpec) [][]interface{} {
	return splitStringSliceToInterface(ipset)
}

func splitDomainNameSetToInterface(set db.DomainNameSetSpec) [][]interface{} {
	return splitStringSliceToInterface(set)
}

func splitStringSliceToInterface(set []string) [][]interface{} {
	terms := make([][]interface{}, 1)
	for _, t := range set {
		if len(terms[len(terms)-1]) >= MaxClauseCount {
			terms = append(terms, []interface{}{t})
		} else {
			terms[len(terms)-1] = append(terms[len(terms)-1], t)
		}
	}
	return terms

}

func (e *Elastic) DeleteIPSet(ctx context.Context, m db.Meta) error {
	return e.deleteSet(ctx, m, IPSetIndex)
}

func (e *Elastic) DeleteDomainNameSet(ctx context.Context, m db.Meta) error {
	return e.deleteSet(ctx, m, DomainNameSetIndex)
}

func (e *Elastic) deleteSet(ctx context.Context, m db.Meta, idx string) error {
	ds := e.c.Delete().Index(idx).Id(m.Name)
	if m.Version != nil {
		ds = ds.Version(*m.Version)
	}
	_, err := ds.Do(ctx)
	return err
}

func (e *Elastic) PutSecurityEvent(ctx context.Context, f db.SecurityEventInterface) error {
	// Wait for the SecurityEvent Mapping to be created
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(CreateIndexWaitTimeout):
		return errors.New("Timeout waiting for index creation")
	case <-e.eventMappingCreated:
		break
	}
	_, err := e.c.Index().Index(EventIndex).Id(f.ID()).BodyJson(f).Do(ctx)
	return err
}

func (e *Elastic) GetDatafeeds(ctx context.Context, feedIDs ...string) ([]DatafeedSpec, error) {
	params := strings.Join(feedIDs, ",")

	resp, err := e.c.PerformRequest(ctx, elastic.PerformRequestOptions{
		Method: "GET",
		Path:   fmt.Sprintf("/_ml/datafeeds/%s", params),
	})
	if err != nil {
		return nil, err
	}

	var getDatafeedsResponse GetDatafeedResponseSpec
	err = json.Unmarshal(resp.Body, &getDatafeedsResponse)
	if err != nil {
		return nil, err
	}

	return getDatafeedsResponse.Datafeeds, nil
}

func (e *Elastic) GetDatafeedStats(ctx context.Context, feedIDs ...string) ([]DatafeedCountsSpec, error) {
	params := strings.Join(feedIDs, ",")

	resp, err := e.c.PerformRequest(ctx, elastic.PerformRequestOptions{
		Method: "GET",
		Path:   fmt.Sprintf("/_ml/datafeeds/%s/_stats", params),
	})
	if err != nil {
		return nil, err
	}

	var getDatafeedStatsResponse GetDatafeedStatsResponseSpec
	err = json.Unmarshal(resp.Body, &getDatafeedStatsResponse)
	if err != nil {
		return nil, err
	}

	return getDatafeedStatsResponse.Datafeeds, nil
}

func (e *Elastic) StartDatafeed(ctx context.Context, feedID string, options *OpenDatafeedOptions) (bool, error) {
	requestOptions := elastic.PerformRequestOptions{
		Method: "POST",
		Path:   fmt.Sprintf("/_ml/datafeeds/%s/_start", feedID),
	}
	if options != nil {
		requestOptions.Body = options
	}
	resp, err := e.c.PerformRequest(ctx, requestOptions)
	if err != nil {
		return false, err
	}

	var openJobResponse map[string]bool
	err = json.Unmarshal(resp.Body, &openJobResponse)
	if err != nil {
		return false, err
	}

	return openJobResponse["started"], nil
}

func (e *Elastic) StopDatafeed(ctx context.Context, feedID string, options *CloseDatafeedOptions) (bool, error) {
	requestOptions := elastic.PerformRequestOptions{
		Method: "POST",
		Path:   fmt.Sprintf("/_ml/datafeeds/%s/_stop", feedID),
	}
	if options != nil {
		requestOptions.Body = options
	}
	resp, err := e.c.PerformRequest(ctx, requestOptions)
	if err != nil {
		return false, err
	}

	var openDatafeedResponse map[string]bool
	err = json.Unmarshal(resp.Body, &openDatafeedResponse)
	if err != nil {
		return false, err
	}

	return openDatafeedResponse["stopped"], nil
}

func (e *Elastic) GetJobs(ctx context.Context, jobIDs ...string) ([]JobSpec, error) {
	params := strings.Join(jobIDs, ",")

	resp, err := e.c.PerformRequest(ctx, elastic.PerformRequestOptions{
		Method: "GET",
		Path:   fmt.Sprintf("/_ml/anomaly_detectors/%s", params),
	})
	if err != nil {
		return nil, err
	}

	var getJobsResponse GetJobResponseSpec
	err = json.Unmarshal(resp.Body, &getJobsResponse)
	if err != nil {
		return nil, err
	}

	return getJobsResponse.Jobs, nil
}

func (e *Elastic) GetJobStats(ctx context.Context, jobIDs ...string) ([]JobStatsSpec, error) {
	params := strings.Join(jobIDs, ",")

	resp, err := e.c.PerformRequest(ctx, elastic.PerformRequestOptions{
		Method: "GET",
		Path:   fmt.Sprintf("/_ml/anomaly_detectors/%s/_stats", params),
	})
	if err != nil {
		return nil, err
	}

	var getJobsStatsResponse GetJobStatsResponseSpec
	err = json.Unmarshal(resp.Body, &getJobsStatsResponse)
	if err != nil {
		return nil, err
	}

	return getJobsStatsResponse.Jobs, nil
}

func (e *Elastic) OpenJob(ctx context.Context, jobID string, options *OpenJobOptions) (bool, error) {
	requestOptions := elastic.PerformRequestOptions{
		Method: "POST",
		Path:   fmt.Sprintf("/_ml/anomaly_detectors/%s/_open", jobID),
	}
	if options != nil {
		requestOptions.Body = options
	}
	resp, err := e.c.PerformRequest(ctx, requestOptions)
	if err != nil {
		return false, err
	}

	var openJobResponse map[string]bool
	err = json.Unmarshal(resp.Body, &openJobResponse)
	if err != nil {
		return false, err
	}

	return openJobResponse["opened"], nil
}

func (e *Elastic) CloseJob(ctx context.Context, jobID string, options *CloseJobOptions) (bool, error) {
	requestOptions := elastic.PerformRequestOptions{
		Method: "POST",
		Path:   fmt.Sprintf("/_ml/anomaly_detectors/%s/_close", jobID),
	}
	if options != nil {
		requestOptions.Body = options
	}
	resp, err := e.c.PerformRequest(ctx, requestOptions)
	if err != nil {
		return false, err
	}

	var openJobResponse map[string]bool
	err = json.Unmarshal(resp.Body, &openJobResponse)
	if err != nil {
		return false, err
	}

	return openJobResponse["closed"], nil
}

func (e *Elastic) GetBuckets(ctx context.Context, jobID string, options *GetBucketsOptions) ([]BucketSpec, error) {
	optTimestamp := ""
	if options.Timestamp != nil {
		optTimestamp = fmt.Sprintf("/%s", options.Timestamp.Format(time.RFC3339))
	}

	requestOptions := elastic.PerformRequestOptions{
		Method: "POST",
		Path:   fmt.Sprintf("/_ml/anomaly_detectors/%s/results/buckets%s", jobID, optTimestamp),
	}
	if options != nil {
		requestOptions.Body = options
	}
	resp, err := e.c.PerformRequest(ctx, requestOptions)
	if err != nil {
		return nil, err
	}

	var getBucketsResponse GetBucketsResponseSpec
	err = json.Unmarshal(resp.Body, &getBucketsResponse)
	if err != nil {
		return nil, err
	}

	return getBucketsResponse.Buckets, nil
}

func (e *Elastic) GetRecords(ctx context.Context, jobID string, options *GetRecordsOptions) ([]RecordSpec, error) {
	requestOptions := elastic.PerformRequestOptions{
		Method: "POST",
		Path:   fmt.Sprintf("/_ml/anomaly_detectors/%s/results/records", jobID),
	}
	if options != nil {
		requestOptions.Body = options
	}
	resp, err := e.c.PerformRequest(ctx, requestOptions)
	if err != nil {
		return nil, err
	}

	var getRecordsResponse GetRecordsResponseSpec
	err = json.Unmarshal(resp.Body, &getRecordsResponse)
	if err != nil {
		return nil, err
	}

	return getRecordsResponse.Records, nil
}

func (e *Elastic) ObjectCreatedBetween(
	ctx context.Context, resource, namespace, name string, before, after time.Time,
) (bool, error) {
	return e.auditObjectCreatedDeletedBetween(ctx, Create, resource, namespace, name, before, after)
}

func (e *Elastic) ObjectDeletedBetween(
	ctx context.Context, resource, namespace, name string, before, after time.Time,
) (bool, error) {
	return e.auditObjectCreatedDeletedBetween(ctx, Delete, resource, namespace, name, before, after)
}

func (e *Elastic) auditObjectCreatedDeletedBetween(
	ctx context.Context,
	verb, resource, namespace, name string,
	before, after time.Time,
) (bool, error) {
	switch {
	case verb == "":
		panic("missing verb parameter")
	case resource == "":
		panic("missing resource parameter")
	case name == "":
		return false, errors.New("missing name parameter")
	}

	// Build query using given fields.
	queries := []elastic.Query{
		elastic.NewRangeQuery("stageTimestamp").Gte(after).Lte(before),
		elastic.NewMatchQuery("verb", verb),
		elastic.NewMatchQuery("objectRef.resource", resource),
		elastic.NewMatchQuery("objectRef.name", name),
	}

	if namespace != "" {
		queries = append(queries, elastic.NewMatchQuery("objectRef.namespace", namespace))
	}

	query := elastic.NewBoolQuery().Filter(queries...)

	// Get the number of matching entries.
	result, err := elastic.NewSearchService(e.c).Index(AuditIndex).Size(AuditQuerySize).Query(query).Do(ctx)
	if err != nil {
		return false, err
	}

	rval := result.TotalHits() > 0

	log.WithFields(log.Fields{
		"verb":      verb,
		"resource":  resource,
		"namespace": namespace,
		"name":      name,
		"before":    fmt.Sprint(before),
		"after":     fmt.Sprint(after),
		"totalHits": result.TotalHits(),
		"found":     rval,
	}).Debug("AuditLog query results")

	return rval, nil
}

func (e *Elastic) ListWatches(ctx context.Context) ([]db.Meta, error) {
	result, err := e.c.Search(WatchIndex).Query(elastic.NewMatchAllQuery()).Do(ctx)

	// Handle the special case where no watches have been created. Querying for watches returns
	// index_not_found_exception. This is not an error. There are no watches.
	if err != nil {
		switch err.(type) {
		case *elastic.Error:
			err := err.(*elastic.Error)
			if err.Details.Type == "index_not_found_exception" {
				return nil, nil
			}
		}
		return nil, err
	}

	var res []db.Meta
	for _, hit := range result.Hits.Hits {
		if strings.HasPrefix(hit.Id, WatchNamePrefix) {
			res = append(res, db.Meta{Name: hit.Id[len(WatchNamePrefix):], Version: hit.Version})
		}
	}
	return res, nil
}

func (e *Elastic) ExecuteWatch(ctx context.Context, body *ExecuteWatchBody) (*elastic.XPackWatchRecord, error) {
	res, err := e.c.XPackWatchExecute().BodyJson(body).Do(ctx)

	if res != nil {
		return res.WatchRecord, err
	}
	return nil, err
}

func (e *Elastic) PutWatch(ctx context.Context, name string, body *PutWatchBody) error {
	watchID := WatchNamePrefix + name
	_, err := e.c.XPackWatchPut(watchID).Body(body).Do(ctx)
	return err
}

func (e *Elastic) GetWatchStatus(ctx context.Context, name string) (*elastic.XPackWatchStatus, error) {
	res, err := e.c.XPackWatchGet(WatchNamePrefix + name).Do(ctx)
	if err != nil {
		return nil, err
	}
	return res.Status, err
}

func (e *Elastic) DeleteWatch(ctx context.Context, m db.Meta) error {
	// TODO This has a race condition. :(
	// should maybe check version before or in result. Elastic does not allow you to specify version which
	// causes the race.
	watchID := WatchNamePrefix + m.Name
	_, err := e.c.XPackWatchDelete(watchID).Do(ctx)
	return err
}
