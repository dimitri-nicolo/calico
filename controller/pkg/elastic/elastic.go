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
	"time"

	"github.com/araddon/dateparse"
	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/events"
)

const (
	IPSetIndexPattern   = ".tigera.ipset.%s"
	StandardType        = "_doc"
	FlowLogIndexPattern = "tigera_secure_ee_flows.%s.*"
	EventIndexPattern   = "tigera_secure_ee_events.%s"
	QuerySize           = 1000
	MaxClauseCount      = 1024
)

var IPSetIndex string
var EventIndex string
var FlowLogIndex string

func init() {
	cluster := os.Getenv("CLUSTER_NAME")
	if cluster == "" {
		cluster = "cluster"
	}
	IPSetIndex = fmt.Sprintf(IPSetIndexPattern, cluster)
	EventIndex = fmt.Sprintf(EventIndexPattern, cluster)
	FlowLogIndex = fmt.Sprintf(FlowLogIndexPattern, cluster)
}

type ipSetDoc struct {
	CreatedAt time.Time    `json:"created_at"`
	IPs       db.IPSetSpec `json:"ips"`
}

type Elastic struct {
	c *elastic.Client
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
	return &Elastic{c}, nil
}

func (e *Elastic) ListIPSets(ctx context.Context) ([]db.IPSetMeta, error) {
	q := elastic.NewMatchAllQuery()
	scroller := e.c.Scroll(IPSetIndex).Type(StandardType).Version(true).FetchSource(false).Query(q)

	var ids []db.IPSetMeta
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
			ids = append(ids, db.IPSetMeta{Name: hit.Id, Version: hit.Version})
		}
	}
}

func (e *Elastic) PutIPSet(ctx context.Context, name string, set db.IPSetSpec) error {
	err := e.ensureIndexExists(ctx, IPSetIndex, ipSetMapping)
	if err != nil {
		return err
	}

	// Put document
	body := ipSetDoc{CreatedAt: time.Now(), IPs: set}
	_, err = e.c.Index().Index(IPSetIndex).Type(StandardType).Id(name).BodyJson(body).Do(ctx)
	log.WithField("name", name).Info("IP set stored")

	return err
}

func (e *Elastic) ensureIndexExists(ctx context.Context, idx, mapping string) error {
	// Ensure Index exists
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
	res, err := e.c.Get().Index(IPSetIndex).Type(StandardType).Id(name).Do(ctx)
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
	res, err := e.c.Get().Index(IPSetIndex).Type(StandardType).Id(name).FetchSourceContext(elastic.NewFetchSourceContext(true).Include("created_at")).Do(ctx)
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

func (e *Elastic) DeleteIPSet(ctx context.Context, m db.IPSetMeta) error {
	ds := e.c.Delete().Index(IPSetIndex).Type(StandardType).Id(m.Name)
	if m.Version != nil {
		ds = ds.Version(*m.Version)
	}
	_, err := ds.Do(ctx)
	return err
}

func (e *Elastic) PutSecurityEvent(ctx context.Context, f events.SecurityEvent) error {
	err := e.ensureIndexExists(ctx, EventIndex, eventMapping)
	if err != nil {
		return err
	}

	_, err = e.c.Index().Index(EventIndex).Type(StandardType).Id(f.ID()).BodyJson(f).Do(ctx)
	return err
}
