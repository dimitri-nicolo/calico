// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package templates

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/olivere/elastic/v7"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

// Load is the callback signature used to load values in the cache. The default implementation is IndexBootstrapper
type Load func(ctx context.Context, client *elastic.Client, config *TemplateConfig) (*Template, error)

type templateCache struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client
	shards    int
	replicas  int

	sync.RWMutex
	cache map[string]*Template
	load  Load
}

func NewTemplateCache(c lmaelastic.Client, shards, replicas int) bapi.Cache {
	return &templateCache{
		client:    c.Backend(),
		lmaclient: c,
		shards:    shards,
		replicas:  replicas,

		cache: make(map[string]*Template),
		load:  IndexBootstrapper,
	}
}

func (b *templateCache) InitializeIfNeeded(ctx context.Context, logsType bapi.DataType, info bapi.ClusterInfo) error {
	ok := b.getEntry(logsType, info)
	if !ok {
		return b.loadEntry(ctx, logsType, info)
	}

	return nil
}

func (b *templateCache) getEntry(logsType bapi.DataType, info bapi.ClusterInfo) bool {
	b.RLock()
	defer b.RUnlock()

	key := b.buildKey(logsType, info)
	_, ok := b.cache[key]

	return ok
}

func (b *templateCache) loadEntry(ctx context.Context, logsType bapi.DataType, info bapi.ClusterInfo) error {
	b.Lock()
	defer b.Unlock()

	templateConfig := NewTemplateConfig(logsType, info,
		WithShards(b.shards),
		WithReplicas(b.replicas))

	template, err := b.load(ctx, b.client, templateConfig)
	if err != nil {
		return err
	}

	key := b.buildKey(logsType, info)
	b.cache[key] = template

	return nil
}

func (b *templateCache) buildKey(logsType bapi.DataType, info bapi.ClusterInfo) string {
	if info.Tenant == "" {
		return fmt.Sprintf("%s-%s", strings.ToLower(string(logsType)), info.Cluster)
	}

	return fmt.Sprintf("%s-%s-%s", strings.ToLower(string(logsType)), info.Cluster, info.Tenant)
}
