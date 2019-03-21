// Copyright 2019 Tigera Inc. All rights reserved.

package elastic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/araddon/dateparse"
	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/events"
)

const IPSetIndex = ".tigera.ipset"
const StandardType = "_doc"
const FlowLogIndex = "tigera_secure_ee_flows*"
const EventIndex = "tigera_secure_ee_events"
const QuerySize = 1000

type ipSetDoc struct {
	CreatedAt time.Time    `json:"created_at"`
	IPs       db.IPSetSpec `json:"ips"`
}

type Elastic struct {
	c *elastic.Client
}

func NewElastic(h *http.Client, url *url.URL, username, password string) *Elastic {

	options := []elastic.ClientOptionFunc{
		elastic.SetURL(url.String()),
		elastic.SetHttpClient(h),
		elastic.SetErrorLog(log.StandardLogger()),
		elastic.SetSniff(false),
		//elastic.SetTraceLog(log.StandardLogger()),
	}
	if username != "" {
		options = append(options, elastic.SetBasicAuth(username, password))
	}
	c, err := elastic.NewClient(options...)
	if err != nil {
		panic(err)
	}
	return &Elastic{c}
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
	f := func(ipset, field string) *elastic.ScrollService {
		q := elastic.NewTermsQuery(field).TermsLookup(
			elastic.NewTermsLookup().
				Index(IPSetIndex).
				Type(StandardType).
				Id(ipset).
				Path("ips"))
		return e.c.Scroll(FlowLogIndex).SortBy(elastic.SortByDoc{}).Query(q).Size(QuerySize)
	}

	return &elasticFlowLogIterator{
		scrollers: map[string]Scroller{"source_ip": f(name, "source_ip"), "dest_ip": f(name, "dest_ip")},
		ctx:       ctx,
		name:      name,
	}, nil
}

func (e *Elastic) PutSecurityEvent(ctx context.Context, f events.SecurityEvent) error {
	err := e.ensureIndexExists(ctx, EventIndex, eventMapping)
	if err != nil {
		return err
	}

	_, err = e.c.Index().Index(EventIndex).Type(StandardType).Id(f.ID()).BodyJson(f).Do(ctx)
	return err
}
