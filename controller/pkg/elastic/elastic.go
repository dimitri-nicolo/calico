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
	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
)

const (
	IPSetIndexPattern         = ".tigera.ipset.%s"
	DomainNameSetIndexPattern = ".tigera.domainnameset.%s"
	StandardType              = "_doc"
	FlowLogIndexPattern       = "tigera_secure_ee_flows.%s.*"
	EventIndexPattern         = "tigera_secure_ee_events.%s"
	AuditIndexPattern         = "tigera_secure_ee_audit_*.%s.*"
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
	EventIndex    string
	FlowLogIndex  string
	AuditIndex    string
	IndexByKind   map[db.Kind]string
	MappingByKind map[db.Kind]string
)

func init() {
	cluster := os.Getenv("CLUSTER_NAME")
	if cluster == "" {
		cluster = "cluster"
	}
	ipSetIndex := fmt.Sprintf(IPSetIndexPattern, cluster)
	domainNameSetIndex := fmt.Sprintf(DomainNameSetIndexPattern, cluster)
	EventIndex = fmt.Sprintf(EventIndexPattern, cluster)
	FlowLogIndex = fmt.Sprintf(FlowLogIndexPattern, cluster)
	AuditIndex = fmt.Sprintf(FlowLogIndexPattern, cluster)
	IndexByKind = map[db.Kind]string{
		db.KindIPSet:         ipSetIndex,
		db.KindDomainNameSet: domainNameSetIndex,
	}
	MappingByKind = map[db.Kind]string{
		db.KindIPSet:         ipSetMapping,
		db.KindDomainNameSet: domainNameSetMapping,
	}
}

type ipSetDoc struct {
	CreatedAt time.Time    `json:"created_at"`
	IPs       db.IPSetSpec `json:"ips"`
}

type domainNameSetDoc struct {
	CreatedAt time.Time            `json:"created_at"`
	Names     db.DomainNameSetSpec `json:"names"`
}

type Elastic struct {
	c                   *elastic.Client
	url                 *url.URL
	setMappingCreated   map[db.Kind]chan struct{}
	eventMappingCreated chan struct{}
	elasticIsAlive      bool
	cancel              context.CancelFunc
	once                sync.Once
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
		c:                   c,
		url:                 url,
		setMappingCreated:   make(map[db.Kind]chan struct{}),
		eventMappingCreated: make(chan struct{}),
	}
	for k := range IndexByKind {
		e.setMappingCreated[k] = make(chan struct{})
	}

	return e, nil
}

func (e *Elastic) Run(ctx context.Context) {
	e.once.Do(func() {
		ctx, e.cancel = context.WithCancel(ctx)
		for k, i := range IndexByKind {
			go func() {
				if err := e.createOrUpdateIndex(ctx, i, MappingByKind[k], e.setMappingCreated[k]); err != nil {
					log.WithError(err).WithFields(log.Fields{
						"index": IndexByKind[k],
					}).Error("Could not create index")
				}
			}()
		}

		go func() {
			if err := e.createOrUpdateIndex(ctx, EventIndex, eventMapping, e.eventMappingCreated); err != nil {
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

func (e *Elastic) Ready() bool {
	select {
	case <-e.eventMappingCreated:
		break
	default:
		return false
	}

	for _, c := range e.setMappingCreated {
		select {
		case <-c:
			break
		default:
			return false
		}
	}
	return e.elasticIsAlive
}

func (e *Elastic) ListSets(ctx context.Context, kind db.Kind) ([]db.Meta, error) {
	q := elastic.NewMatchAllQuery()
	idx := IndexByKind[kind]
	scroller := e.c.Scroll(idx).Type(StandardType).Version(true).FetchSource(false).Query(q)

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
			ids = append(ids, db.Meta{Name: hit.Id, Version: hit.Version, Kind: kind})
		}
	}
}

func (e *Elastic) PutSet(ctx context.Context, meta db.Meta, value interface{}) error {
	var body interface{}
	switch meta.Kind {
	case db.KindIPSet:
		body = ipSetDoc{CreatedAt: time.Now(), IPs: value.(db.IPSetSpec)}
	case db.KindDomainNameSet:
		body = domainNameSetDoc{CreatedAt: time.Now(), Names: value.(db.DomainNameSetSpec)}
	default:
		panic("unknown db.Meta kind " + string(meta.Kind))
	}

	// Wait for the Sets Mapping to be created
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(CreateIndexWaitTimeout):
		return errors.New("Timeout waiting for index creation")
	case <-e.setMappingCreated[meta.Kind]:
		break
	}

	// Put document
	_, err := e.c.Index().Index(IndexByKind[meta.Kind]).Type(StandardType).Id(meta.Name).BodyJson(body).Do(ctx)
	log.WithField("name", meta.Name).Info("IP set stored")

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
	// Ensure Index exists, or update mappings if it does
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
	} else {
		var m map[string]map[string]interface{}
		err := json.Unmarshal([]byte(mapping), &m)
		if err != nil {
			return err
		}

		for k, v := range m["mappings"] {
			b, err := json.Marshal(&v)
			if err != nil {
				return err
			}

			r, err := e.c.PutMapping().Index(idx).Type(k).BodyString(string(b)).Do(ctx)
			if err != nil {
				return err
			}
			if !r.Acknowledged {
				return fmt.Errorf("not acknowledged index %s update", idx)
			}
		}
	}
	return nil
}

func (e *Elastic) GetIPSet(ctx context.Context, name string) (db.IPSetSpec, error) {
	idx := IndexByKind[db.KindIPSet]
	res, err := e.c.Get().Index(idx).Type(StandardType).Id(name).Do(ctx)
	if err != nil {
		return nil, err
	}

	if res.Source == nil {
		return nil, errors.New("Elastic document has nil Source")
	}

	var doc map[string]interface{}
	err = json.Unmarshal(*res.Source, &doc)
	if err != nil {
		return nil, err
	}
	i, ok := doc["ips"]
	if !ok {
		return nil, errors.New("Elastic document missing ips section")
	}

	ia, ok := i.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Unknown type for %#v", i)
	}
	ips := db.IPSetSpec{}
	for _, v := range ia {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("Unknown type for %#v", s)
		}
		ips = append(ips, s)
	}

	return ips, nil
}

func (e *Elastic) GetIPSetModified(ctx context.Context, name string) (time.Time, error) {
	idx := IndexByKind[db.KindIPSet]
	res, err := e.c.Get().Index(idx).Type(StandardType).Id(name).FetchSourceContext(elastic.NewFetchSourceContext(true).Include("created_at")).Do(ctx)
	if err != nil {
		return time.Time{}, err
	}

	if res.Source == nil {
		return time.Time{}, err
	}

	var doc map[string]interface{}
	err = json.Unmarshal(*res.Source, &doc)
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

func (e *Elastic) QueryIPSet(ctx context.Context, name string) (db.SecurityEventIterator, error) {
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
		scrollers = append(scrollers, scrollerEntry{name: "source_ip", scroller: f(name, "source_ip", t), terms: t})
		scrollers = append(scrollers, scrollerEntry{name: "dest_ip", scroller: f(name, "dest_ip", t), terms: t})
	}

	return &flowLogIterator{
		scrollers: scrollers,
		ctx:       ctx,
		name:      name,
	}, nil
}

func splitIPSetToInterface(ipset db.IPSetSpec) [][]interface{} {
	terms := make([][]interface{}, 1)
	for _, ip := range ipset {
		if len(terms[len(terms)-1]) >= MaxClauseCount {
			terms = append(terms, []interface{}{ip})
		} else {
			terms[len(terms)-1] = append(terms[len(terms)-1], ip)
		}
	}
	return terms
}

func (e *Elastic) DeleteSet(ctx context.Context, m db.Meta) error {
	idx := IndexByKind[m.Kind]
	ds := e.c.Delete().Index(idx).Type(StandardType).Id(m.Name)
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
	_, err := e.c.Index().Index(EventIndex).Type(StandardType).Id(f.ID()).BodyJson(f).Do(ctx)
	return err
}

func (e *Elastic) GetDatafeeds(ctx context.Context, feedIDs ...string) ([]DatafeedSpec, error) {
	params := strings.Join(feedIDs, ",")

	resp, err := e.c.PerformRequest(ctx, elastic.PerformRequestOptions{
		Method: "GET",
		Path:   fmt.Sprintf("/_xpack/ml/datafeeds/%s", params),
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
		Path:   fmt.Sprintf("/_xpack/ml/datafeeds/%s/_stats", params),
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
		Path:   fmt.Sprintf("/_xpack/ml/datafeeds/%s/_start", feedID),
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
		Path:   fmt.Sprintf("/_xpack/ml/datafeeds/%s/_stop", feedID),
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
		Path:   fmt.Sprintf("/_xpack/ml/anomaly_detectors/%s", params),
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
		Path:   fmt.Sprintf("/_xpack/ml/anomaly_detectors/%s/_stats", params),
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
		Path:   fmt.Sprintf("/_xpack/ml/anomaly_detectors/%s/_open", jobID),
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
		Path:   fmt.Sprintf("/_xpack/ml/anomaly_detectors/%s/_close", jobID),
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
		Path:   fmt.Sprintf("/_xpack/ml/anomaly_detectors/%s/results/buckets%s", jobID, optTimestamp),
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
		Path:   fmt.Sprintf("/_xpack/ml/anomaly_detectors/%s/results/records", jobID),
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
