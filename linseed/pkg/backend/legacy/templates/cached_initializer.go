// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package templates

import (
	"context"
	"sync"

	"github.com/olivere/elastic/v7"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

// Bootstrapper is a function that initializes an index within Elasticsearch.
type Bootstrapper func(ctx context.Context, client *elastic.Client, config *TemplateConfig) (*Template, error)

// cachedInitializer implements the IndexInitializer interface, using a local cache to avoid
// unnecessary calls to Elasticsearch.
type cachedInitializer struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client
	shards    int
	replicas  int

	sync.RWMutex
	cache map[string]*Template

	// bootstrap is the function used to initialize values in elasticsearch.
	// The default is DefaultBootstrapper, but can be overridden for testing.
	bootstrap Bootstrapper
}

func NewCachedInitializer(c lmaelastic.Client, shards, replicas int) bapi.IndexInitializer {
	return &cachedInitializer{
		client:    c.Backend(),
		lmaclient: c,
		shards:    shards,
		replicas:  replicas,
		cache:     make(map[string]*Template),
		bootstrap: DefaultBootstrapper,
	}
}

func NewNoOpInitializer() bapi.IndexInitializer {
	return &cachedInitializer{
		cache:     make(map[string]*Template),
		bootstrap: NoopBootstrapper,
	}
}

func (i *cachedInitializer) Initialize(ctx context.Context, index bapi.Index, info bapi.ClusterInfo) error {
	if !i.exists(index, info) {
		return i.initialize(ctx, index, info)
	}
	return nil
}

// exists returns whether the index exists in the cache.
func (i *cachedInitializer) exists(index bapi.Index, info bapi.ClusterInfo) bool {
	i.RLock()
	defer i.RUnlock()
	_, ok := i.cache[index.Name(info)]
	return ok
}

// initialize initializes the index in elasticsearch and adds it to the cache.
func (i *cachedInitializer) initialize(ctx context.Context, index bapi.Index, info bapi.ClusterInfo) error {
	i.Lock()
	defer i.Unlock()

	templateConfig := NewTemplateConfig(index, info, WithShards(i.shards), WithReplicas(i.replicas))
	template, err := i.bootstrap(ctx, i.client, templateConfig)
	if err != nil {
		return err
	}

	// Cache the index so that we don't need to re-initialize it on subsequent calls.
	i.cache[index.Name(info)] = template
	return nil
}
