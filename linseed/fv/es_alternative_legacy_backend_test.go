// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"
)

type LegacyBackend struct {
	esClient *elastic.Client
	t        *testing.T
}

func (b *LegacyBackend) ingest(index, doc, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := b.esClient.Index().Index(index).Id(id).BodyJson(doc).Refresh("true").Do(ctx)
	log.Infof("[INGEST DOC] ES response %#v", res)
	require.NoError(b.t, err)
}

func (b *LegacyBackend) delete(index string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := b.esClient.IndexExists(index).Do(ctx)
	require.NoError(b.t, err)

	if exists {
		res, err := b.esClient.DeleteIndex(index).Do(ctx)
		log.Infof("[DELETE] ES response %#v", res)
		require.NoError(b.t, err)
	}
}

func (b *LegacyBackend) ingestFlow() {
	b.ingest("tigera_secure_ee_flows.cluster", flowLog, flowID)
}

func (b *LegacyBackend) deleteFlowLogs() {
	b.delete("tigera_secure_ee_flows.cluster")
}
