package db

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"
)

const IPSetIndex = ".tigera.ipset"
const StandardType = "_doc"
const FlowLogIndex = "tigera_secure_ee_flows*"
const EventIndex = "tigera_secure_ee_events"

const ipSetMapping = `{
  "mappings": {
    "_doc": {
      "properties": {
        "ips": {
            "type": "ip"
        }
      }
    }
  }
}`

const eventMapping = `{
  "mappings": {
    "_doc": {
      "properties" : {
        "start_time": {
            "type": "date",
            "format": "epoch_second"
        },
        "end_time": {
            "type": "date",
            "format": "epoch_second"
        },
        "action": {
            "type": "keyword"
        },
        "bytes_in": {
            "type": "long"
        },
        "bytes_out": {
            "type": "long"
        },
        "dest_ip": {
            "type": "ip",
            "null_value": "0.0.0.0"
        },
        "dest_name": {
            "type": "keyword"
        },
        "dest_name_aggr": {
            "type": "keyword"
        },
        "dest_namespace": {
            "type": "keyword"
        },
        "dest_port": {
            "type": "long",
            "null_value": "0"
        },
        "dest_type": {
            "type": "keyword"
        },
        "dest_labels": {
                /* This is an array of keywords. It is not necessary to declare this as an array. Elastic will automatically accept a list of strings here */
                "type": "nested",
                "properties": {
                        "labels": {"type": "keyword"}
                }
        },
        "reporter": {
            "type": "keyword"
        },
        "num_flows": {
            "type": "long"
        },
        "num_flows_completed": {
            "type": "long"
        },
        "num_flows_started": {
            "type": "long"
        },
        "packets_in": {
            "type": "long"
        },
        "packets_out": {
            "type": "long"
        },
        "proto": {
            "type": "keyword"
        },
        "policies": {
                /* This is an array of keywords. It is not necessary to declare this as an array. Elastic will automatically accept a list of strings here */
                "type": "nested",
                "properties": {
                        "all_policies": {"type": "keyword"}
                }
        },
        "source_ip": {
            "type": "ip",
            "null_value": "0.0.0.0"
        },
        "source_name": {
            "type": "keyword"
        },
        "source_name_aggr": {
            "type": "keyword"
        },
        "source_namespace": {
            "type": "keyword"
        },
        "source_port": {
            "type": "long",
            "null_value": "0"
        },
        "source_type": {
            "type": "keyword"
        },
        "source_labels": {
                /* This is an array of keywords. It is not necessary to declare this as an array. Elastic will automatically accept a list of strings here */
                "type": "nested",
                "properties": {
                        "labels": {"type": "keyword"}
                }
        }
      }   
    }
  }
}`


type ipSetDoc struct {
	IPs []string `json:"ips"`
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

func (e *Elastic) PutIPSet(ctx context.Context, name string, set []string) error {
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

func (e *Elastic) QueryIPSet(ctx context.Context, name string) ([]FlowLog, error) {
	q := elastic.NewTermsQuery("source_ip").TermsLookup(
		elastic.NewTermsLookup().
			Index(IPSetIndex).
			Type(StandardType).
			Id(name).
			Path("ips"))
	r, err := e.c.Search().Index(FlowLogIndex).Query(q).Size(1000).Do(ctx)
	if err != nil {
		return nil, err
	}
	log.WithField("hits", r.TotalHits()).Info("elastic query returned")
	var flows []FlowLog
	for _, hit := range r.Hits.Hits {
		var flow FlowLog
		err := json.Unmarshal(*hit.Source, &flow)
		if err != nil {
			log.WithError(err).WithField("raw", *hit.Source).Error("could not unmarshal")
		}
		flows = append(flows, flow)
	}
	return flows, nil
}

func (e *Elastic) PutFlowLog(ctx context.Context, f FlowLog) error {
	err := e.ensureIndexExists(ctx, EventIndex, eventMapping)
	if err != nil {
		return err
	}

	_, err = e.c.Index().Index(EventIndex).Type(StandardType).Id(f.id()).BodyJson(f).Do(ctx)
	return err
}

func (f FlowLog) id() string {
	return fmt.Sprintf("%d-%s-%s-%s", f.StartTime, f.SourceIP, f.SourceName, f.DestIP)
}
