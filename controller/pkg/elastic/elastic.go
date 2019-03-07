package elastic

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/tigera/intrusion-detection/controller/pkg/flows"

	"github.com/tigera/intrusion-detection/controller/pkg/db"
	"github.com/tigera/intrusion-detection/controller/pkg/feed"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"
)

const IPSetIndex = ".tigera.ipset"
const StandardType = "_doc"
const FlowLogIndex = "tigera_secure_ee_flows*"
const EventIndex = "tigera_secure_ee_events"
const QuerySize = 1000

type ipSetDoc struct {
	IPs feed.IPSet `json:"ips"`
}

type Elastic struct {
	c *elastic.Client
}

func NewElastic(url *url.URL, username, password, pathToCA string) *Elastic {
	ca, err := x509.SystemCertPool()
	if err != nil {
		panic(err)
	}
	if pathToCA != "" {
		cert, err := ioutil.ReadFile(pathToCA)
		if err != nil {
			panic(err)
		}
		ok := ca.AppendCertsFromPEM(cert)
		if !ok {
			panic("failed to add CA")
		}
	}
	h := &http.Client{}
	if url.Scheme == "https" {
		h.Transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: ca}}
	}
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

func (e *Elastic) PutIPSet(ctx context.Context, name string, set feed.IPSet) error {
	err := e.ensureIndexExists(ctx, IPSetIndex, ipSetMapping)
	if err != nil {
		return err
	}

	// Put document
	body := ipSetDoc{set}
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

func (e *Elastic) GetIPSet(name string) ([]string, error) {
	return nil, nil
}

func (e *Elastic) QueryIPSet(ctx context.Context, name string) (db.FlowLogIterator, error) {
	q := elastic.NewTermsQuery("source_ip").TermsLookup(
		elastic.NewTermsLookup().
			Index(IPSetIndex).
			Type(StandardType).
			Id(name).
			Path("ips"))

	return &elasticFlowLogIterator{
		scroll: e.c.Scroll(FlowLogIndex).SortBy(elastic.SortByDoc{}).Query(q).Size(QuerySize),
		ctx:    ctx,
		name:   name,
	}, nil
}

func (e *Elastic) PutFlowLog(ctx context.Context, f flows.FlowLog) error {
	err := e.ensureIndexExists(ctx, EventIndex, eventMapping)
	if err != nil {
		return err
	}

	_, err = e.c.Index().Index(EventIndex).Type(StandardType).Id(f.ID).BodyJson(f).Do(ctx)
	return err
}
