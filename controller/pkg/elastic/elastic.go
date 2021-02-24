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
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/runloop"
	"github.com/tigera/intrusion-detection/controller/pkg/util"
)

const (
	IPSetIndexPattern         = ".tigera.ipset.%s"
	DomainNameSetIndexPattern = ".tigera.domainnameset.%s"
	FlowLogIndexPattern       = "tigera_secure_ee_flows.%s.*"
	DNSLogIndexPattern        = "tigera_secure_ee_dns.%s.*"
	EventIndexPattern         = "tigera_secure_ee_events.%s"
	// EventIndexWildCardPattern is an alternate version of the events index pattern using a wildcard. This is for alert forwarding
	// in a multi-cluster scenario, where we need to query across all cluster event indices.
	EventIndexWildCardPattern   = "tigera_secure_ee_events.*"
	AuditIndexPattern           = "tigera_secure_ee_audit_*.%s.*"
	ForwarderConfigIndexPattern = ".tigera.forwarderconfig.%s"
	WatchIndex                  = ".watches"
	WatchNamePrefixPattern      = "tigera_secure_ee_watch.%s."
	QuerySize                   = 1000
	AuditQuerySize              = 0
	MaxClauseCount              = 1024
	CreateIndexFailureDelay     = time.Second * 15
	CreateIndexWaitTimeout      = time.Minute
	PingTimeout                 = time.Second * 5
	PingPeriod                  = time.Minute
	Create                      = "create"
	Delete                      = "delete"
	DefaultReplicas             = 0
	DefaultShards               = 5
)

var (
	IPSetIndex           string
	DomainNameSetIndex   string
	EventIndex           string
	FlowLogIndex         string
	DNSLogIndex          string
	AuditIndex           string
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
	EventIndex = fmt.Sprintf(EventIndexPattern, clusterName)
	FlowLogIndex = fmt.Sprintf(FlowLogIndexPattern, clusterName)
	DNSLogIndex = fmt.Sprintf(DNSLogIndexPattern, clusterName)
	AuditIndex = fmt.Sprintf(AuditIndexPattern, clusterName)
	WatchNamePrefix = fmt.Sprintf(WatchNamePrefixPattern, clusterName)
	ForwarderConfigIndex = fmt.Sprintf(ForwarderConfigIndexPattern, clusterName)
}

type ipSetDoc struct {
	CreatedAt time.Time    `json:"created_at"`
	IPs       db.IPSetSpec `json:"ips"`
}

type domainNameSetDoc struct {
	CreatedAt time.Time            `json:"created_at"`
	Domains   db.DomainNameSetSpec `json:"domains"`
}

type IndexSettings struct {
	Replicas int `json:"number_of_replicas"`
	Shards   int `json:"number_of_shards"`
}

func DefaultIndexSettings() IndexSettings {
	return IndexSettings{DefaultReplicas, DefaultShards}
}

type Elastic struct {
	c                             *elastic.Client
	url                           *url.URL
	ipSetMappingCreated           chan struct{}
	domainNameSetMappingCreated   chan struct{}
	eventMappingCreated           chan struct{}
	forwarderConfigMappingCreated chan struct{}
	elasticIsAlive                bool
	cancel                        context.CancelFunc
	once                          sync.Once
	indexSettings                 IndexSettings
}

func NewElastic(h *http.Client, url *url.URL, username, password string, indexSettings IndexSettings, debug bool) (*Elastic, error) {
	options := []elastic.ClientOptionFunc{
		elastic.SetURL(url.String()),
		elastic.SetHttpClient(h),
		elastic.SetErrorLog(log.StandardLogger()),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
	}
	// Enable debug (trace-level) logging for the Elastic client library we use
	if debug {
		options = append(options, elastic.SetTraceLog(log.StandardLogger()))
	}
	if username != "" {
		options = append(options, elastic.SetBasicAuth(username, password))
	}
	c, err := elastic.NewClient(options...)
	if err != nil {
		return nil, err
	}
	e := &Elastic{
		c:                             c,
		url:                           url,
		ipSetMappingCreated:           make(chan struct{}),
		domainNameSetMappingCreated:   make(chan struct{}),
		eventMappingCreated:           make(chan struct{}),
		forwarderConfigMappingCreated: make(chan struct{}),
		indexSettings:                 indexSettings,
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

		// We create the index ForwarderConfigIndex regardless of whether the event forwarder is enabled or not (so that it's
		// available when needed).
		go func() {
			if err := e.createOrUpdateIndex(ctx, ForwarderConfigIndex, forwarderConfigMapping, e.forwarderConfigMappingCreated); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"index": ForwarderConfigIndex,
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
	scroller := e.c.Scroll(idx).FetchSource(false).Query(q)
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
			ids = append(ids, db.Meta{
				Name:        hit.Id,
				SeqNo:       hit.SeqNo,
				PrimaryTerm: hit.PrimaryTerm,
			})
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
	if err := util.WaitForChannel(ctx, c, CreateIndexWaitTimeout); err != nil {
		return err
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
		r, err := e.c.CreateIndex(idx).BodyJson(map[string]interface{}{
			"mappings": json.RawMessage(mapping),
			"settings": e.indexSettings,
		}).Do(ctx)
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
	if m.SeqNo != nil {
		ds = ds.IfSeqNo(*m.SeqNo)
	}
	if m.PrimaryTerm != nil {
		ds = ds.IfPrimaryTerm(*m.PrimaryTerm)
	}
	_, err := ds.Do(ctx)
	return err
}

func (e *Elastic) PutSecurityEvent(ctx context.Context, f db.SecurityEventInterface) error {
	// Wait for the SecurityEvent Mapping to be created
	if err := util.WaitForChannel(ctx, e.eventMappingCreated, CreateIndexWaitTimeout); err != nil {
		return err
	}
	_, err := e.c.Index().Index(EventIndex).Id(f.ID()).BodyJson(f).Do(ctx)
	return err
}

// GetSecurityEvents retrieves a listing of security events from ES sorted in ascending order,
// where each events time falls within the range given by start and end time.
func (e *Elastic) GetSecurityEvents(ctx context.Context, start, end time.Time, allClusters bool) ([]db.SecurityEvent, error) {
	l := log.WithFields(logrus.Fields{"func": "GetSecurityEvents"})

	// Determine whether we're querying for events across all clusters (or just the current cluster)
	index := EventIndex
	if allClusters {
		index = EventIndexWildCardPattern
	}

	searchResult, err := e.c.Search().
		Index(index).
		Query(elastic.NewRangeQuery("time").Gte(start).Lte(end)). // query for events within the specified time range
		Sort("time", true).                                       // sort by "time" field, ascending order (oldest events first)
		From(0).Size(10000).                                      // we want all events from the time range (avoid pagination)
		Do(ctx)                                                   // execute

	if err != nil {
		return nil, err
	}

	l.Debugf("Query index %s took %d milliseconds", index, searchResult.TookInMillis)
	l.Debugf("Found a total of %d security events", searchResult.TotalHits())

	var events []db.SecurityEvent
	if searchResult.Hits.TotalHits.Value > 0 {
		// We are concerned mainly with getting the raw JSON of each event
		for _, hit := range searchResult.Hits.Hits {
			l.Debugf("Adding security event with id[%s]", hit.Id)
			events = append(events, db.SecurityEvent{
				Data: hit.Source,
				ID:   hit.Id,
			})
		}
	}

	return events, nil
}

// PutForwarderConfig saves the given ForwarderConfig object back to the datastore.
func (e *Elastic) PutForwarderConfig(ctx context.Context, id string, f *db.ForwarderConfig) error {
	l := log.WithFields(logrus.Fields{"func": "PutForwarderConfig"})
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
func (e *Elastic) GetForwarderConfig(ctx context.Context, id string) (*db.ForwarderConfig, error) {
	l := log.WithFields(logrus.Fields{"func": "GetForwarderConfig"})
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
		for _, hit := range searchResult.Hits.Hits {
			l.Debugf("Selecting forwarder config with id[%s]", hit.Id)
			var config db.ForwarderConfig
			err := json.Unmarshal(hit.Source, &config)
			if err != nil {
				return nil, err
			}
			return &config, nil
		}
	}

	return nil, nil
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
			res = append(res, db.Meta{
				Name:        hit.Id[len(WatchNamePrefix):],
				SeqNo:       hit.SeqNo,
				PrimaryTerm: hit.PrimaryTerm,
			})
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
	// Wait for the SecurityEvent Mapping to be created
	if err := util.WaitForChannel(ctx, e.eventMappingCreated, CreateIndexWaitTimeout); err != nil {
		return err
	}

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
