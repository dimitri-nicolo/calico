package rule

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	model "github.com/tigera/honeypod-recommendation/pkg/model"
	"github.com/tigera/lma/pkg/elastic"
)

const (
	AuditLogProcessorType = "Audit"
	AlertLogProcessorType = "Alert"
	DNSLogProcessorType   = "DNS"
	ADLogProcessorType    = "AnomalyDetection"
)

type ElasticProcessorCfg struct {
	Name    string        `yaml:"name"`
	Type    string        `yaml:"type"`
	Spec    interface{}   `yaml:"spec"`
	ViewMap model.ViewMap `yaml:"viewMap"`
}

func NewElasticProcessor(cfg ElasticProcessorCfg,
	c elastic.Client,
	ctx context.Context,
	rule *Rule,
) (Processor, error) {
	var p Processor
	var err error
	switch cfg.Type {
	case AuditLogProcessorType:
		if p, err = NewAuditLogProcessor(cfg, c, ctx, rule); err != nil {
			return nil, fmt.Errorf("Error constructing audit log processor: %v", err)
		}
	case AlertLogProcessorType:
		if p, err = NewAlertLogProcessor(cfg, c, ctx, rule); err != nil {
			return nil, fmt.Errorf("Error constructing alert log processor: %v", err)
		}
	case DNSLogProcessorType:
		if p, err = NewDNSLogProcessor(cfg, c, ctx, rule); err != nil {
			return nil, fmt.Errorf("Error constructing DNS log processor: %v", err)
		}
	case ADLogProcessorType:
		if p, err = NewADLogProcessor(cfg, c, ctx, rule); err != nil {
			return nil, fmt.Errorf("Error constructing anomaly detection log processor: %v", err)
		}
	default:
		return nil, fmt.Errorf("Unrecognized elasticsearch processor type: %s", cfg.Type)
	}
	return p, nil
}

func (p *ElasticProcessorCfg) UnmarshalYAML(unmarshal func(interface{}) error) error {
	ts := struct {
		Name    string        `yaml:"name"`
		Type    string        `yaml:"type"`
		ViewMap model.ViewMap `yaml:"viewMap"`
	}{}
	err := unmarshal(&ts)
	if err != nil {
		return err
	}
	p.Name = ts.Name
	p.Type = ts.Type
	p.ViewMap = ts.ViewMap
	switch ts.Type {
	case AuditLogProcessorType:
		ss := struct {
			Spec AuditLogProcessorSpec `yaml:"spec"`
		}{}
		err = unmarshal(&ss)
		p.Spec = ss.Spec
	case AlertLogProcessorType:
		ss := struct {
			Spec AlertLogProcessorSpec `yaml:"spec"`
		}{}
		err = unmarshal(&ss)
		p.Spec = ss.Spec
	case DNSLogProcessorType:
		ss := struct {
			Spec DNSLogProcessorSpec `yaml:"spec"`
		}{}
		err = unmarshal(&ss)
		p.Spec = ss.Spec
	case ADLogProcessorType:
		ss := struct {
			Spec ADLogProcessorSpec `yaml:"spec"`
		}{}
		err = unmarshal(&ss)
		p.Spec = ss.Spec
	default:
		log.WithField("type", p.Type).Fatal("Unrecognized processing type")
	}
	return nil
}
