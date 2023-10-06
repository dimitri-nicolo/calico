// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package events

import (
	"context"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/projectcalico/calico/libcalico-go/lib/validator/v3/query"
	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	lsClient "github.com/projectcalico/calico/linseed/pkg/client"
	lsRest "github.com/projectcalico/calico/linseed/pkg/client/rest"
	lmaV1 "github.com/projectcalico/calico/lma/pkg/apis/v1"
	"github.com/sirupsen/logrus"
)

var securityEventsClient lsClient.EventsInterface

type LinseedCfg struct {
	Cluster    string `envconfig:"LINSEED_CLUSTER" default:"cluster"`
	TenantId   string `envconfig:"LINSEED_TENANT_ID"`
	URL        string `envconfig:"LINSEED_URL" default:"https://tigera-linseed.tigera-elasticsearch.svc"`
	CA         string `envconfig:"LINSEED_CA" default:"/etc/pki/tls/certs/tigera-ca-bundle.crt"`
	ClientCert string `envconfig:"LINSEED_CLIENT_CERT" default:"/etc/pki/tls/certs/tigera-ca-bundle.crt"`
	ClientKey  string `envconfig:"LINSEED_CLIENT_KEY"`
	Token      string `envconfig:"LINSEED_TOKEN" default:"/var/run/secrets/kubernetes.io/serviceaccount/token"`
}

func init() {
	config := new(LinseedCfg)
	envconfig.MustProcess("linseed", config)

	client, err := lsClient.NewClient(
		config.TenantId,
		lsRest.Config{
			URL:            config.URL,
			CACertPath:     config.CA,
			ClientCertPath: config.ClientCert,
			ClientKeyPath:  config.ClientKey,
		},
		lsRest.WithTokenPath(config.Token),
	)

	if err == nil {
		securityEventsClient = client.Events(config.Cluster)
		logrus.Info("Linseed connection initialized")
	} else {
		logrus.WithError(err).Fatal(("Linseed connection error"))
	}
}

func FetchSecurityEventsFunc(ctx context.Context, query *query.Query, fromStamp time.Time, toStamp time.Time) []lsApi.Event {
	logrus.WithField("query", query.String()).Info("Fetching security events from Linseed")

	queryParameters := lsApi.EventParams{
		QueryParams: lsApi.QueryParams{
			TimeRange: &lmaV1.TimeRange{
				From: fromStamp,
				To:   toStamp,
			},
		},
		LogSelectionParams: lsApi.LogSelectionParams{
			Selector: query.String(),
		},
	}

	events, err := securityEventsClient.List(ctx, &queryParameters)
	if err != nil {
		logrus.Error("Linseed error occured when fetching events")
		return []lsApi.Event{}
	}

	return events.Items
}
