package elastic

import "sync"

// ClusterContextClientFactory is an Elasticsearch client factory interface whose implementors create Elasticsearch clients
// who are restricted to accessing Elasticsearch data only for the specific cluster given.
type ClusterContextClientFactory interface {
	ClientForCluster(string) (Client, error)
}

// clientFactory is an implementation of the ClusterContextClientFactory. Elasticsearch clients returned by ClientForCluster
// will have the ElasticIndexSuffix of their config set to the given clusterID.
//
// Note that this factory does not cache the clients. The reasoning behind this is that it is not clear how many clusters
// will be connect and how long these clusters will live for, and without extra logic there is no way to clean out the outdated
// clients for clusters that no longer exist. The responsibility of whether a client is held is is the responsibility of
// the caller, who is expected to hold the client and not create request a new client from this factory if they want a long
// lived client.
type clientFactory struct {
	sync.Mutex
	baseConfig *Config
}

func NewClusterContextClientFactory(config *Config) ClusterContextClientFactory {
	return &clientFactory{
		baseConfig: config,
	}
}

// ClientForCluster creates an returns an a Client used to interact with a specific clusters indices
func (f *clientFactory) ClientForCluster(clusterID string) (Client, error) {
	cfg := f.copyConfig()
	cfg.ElasticIndexSuffix = clusterID

	return NewFromConfig(cfg)
}

func (f *clientFactory) copyConfig() *Config {
	f.Lock()
	defer f.Unlock()

	return CopyConfig(f.baseConfig)
}
