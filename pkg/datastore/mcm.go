package datastore

import (
	"github.com/tigera/compliance/pkg/config"
	"net/http"
	"os"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/apiserver/pkg/client/clientset_generated/clientset"
	v3 "github.com/tigera/apiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	"github.com/tigera/lma/pkg/auth"
	"github.com/tigera/lma/pkg/elastic"
)

const (
	// Default cluster name for standalone and management cluster.
	DefaultCluster    = "cluster"
	XClusterIDHeader  = "x-cluster-id"
)

// Bundles clientMap behind convenience methods for multi cluster setups.
type RESTClientFactory interface {
	CalicoClient(clusterID string) v3.ProjectcalicoV3Interface
	ESClient(clusterID string) elastic.Client
	K8sAuth(clusterID string) auth.K8sAuthInterface
}

// Loads current cluster eagerly, loads external clusters lazily.
type RESTClientHolder struct {
	clientMap map[string]clusterClients
	sync.Mutex
	multiClusterForwardingCA       string
	multiClusterForwardingEndpoint string
}

// Groups clients per cluster
type clusterClients struct {
	calico v3.ProjectcalicoV3Interface
	es     elastic.Client
	k8s    auth.K8sAuthInterface
}

// Creates an instance of RESTClientFactory, which holds various clients.
// Loads current cluster eagerly, loads external clusters lazily.
func MustGetRESTClient(config *config.Config) RESTClientFactory {
	client := RESTClientHolder{}
	client.multiClusterForwardingCA = config.MultiClusterForwardingCA
	client.multiClusterForwardingEndpoint = config.MultiClusterForwardingEndpoint
	client.clientMap = map[string]clusterClients{
		DefaultCluster: {
			MustGetCalicoClient(),
			elastic.MustGetElasticClient(),
			auth.NewK8sAuth(MustGetKubernetesClient(), MustGetConfig()),
		},
	}
	return &client
}

// Return a client that lets you fetch calico objects from the datastore for a given cluster.
func (c *RESTClientHolder) CalicoClient(clusterID string) v3.ProjectcalicoV3Interface {
	c.Lock()
	defer c.Unlock()
	if clusterID == "" {
		return c.clientMap[DefaultCluster].calico
	}
	if _, ok := c.clientMap[clusterID]; !ok {
		c.mustAddCluster(clusterID, c.multiClusterForwardingCA, c.multiClusterForwardingEndpoint)
	}
	return c.clientMap[clusterID].calico
}

// Return a client that lets you fetch reports from elasticsearch for a given cluster.
func (c *RESTClientHolder) ESClient(clusterID string) elastic.Client {
	c.Lock()
	defer c.Unlock()
	if clusterID == "" {
		return c.clientMap[DefaultCluster].es
	}
	if _, ok := c.clientMap[clusterID]; !ok {
		c.mustAddCluster(clusterID, c.multiClusterForwardingCA, c.multiClusterForwardingEndpoint)
	}
	return c.clientMap[clusterID].es
}

// Return a client that lets you check authn and authz for a given cluster.
func (c *RESTClientHolder) K8sAuth(clusterID string) auth.K8sAuthInterface {
	c.Lock()
	defer c.Unlock()
	if clusterID == "" {
		return c.clientMap[DefaultCluster].k8s
	}
	if _, ok := c.clientMap[clusterID]; !ok {
		c.mustAddCluster(clusterID, c.multiClusterForwardingCA, c.multiClusterForwardingEndpoint)
	}
	return c.clientMap[clusterID].k8s
}

// Lazily add clients for a new cluster.
func (c *RESTClientHolder) mustAddCluster(clusterID, caPath, host string) {
	log.Infof("Creating new cluster clientMap for clusterID '%s'", clusterID)
	cfg := mustCreateClusterConfig(clusterID, caPath, host)
	// Build k8s client
	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.WithError(err).Panic("Failed to load k8s client")
	}

	// Build calico client
	client, err := clientset.NewForConfig(cfg)
	if err != nil {
		log.WithError(err).Panic("Failed to load Calico client")
	}

	c.clientMap[clusterID] = clusterClients{
		client.ProjectcalicoV3(),
		mustCreateESClientForCluster(clusterID),
		auth.NewK8sAuth(k8sClient, cfg),
	}
}

// createCalicoClientForCluster returns an ES client that queries target cluster indices only.
func mustCreateESClientForCluster(clusterID string) elastic.Client {
	config := elastic.MustLoadConfig()
	config.ElasticIndexSuffix = clusterID
	// Build calico client
	c, err := elastic.NewFromConfig(config)
	if err != nil {
		log.WithError(err).Panicf("Unable to connect to Elasticsearch")
	}
	return c
}

// Create a config that routes requests through Voltron to a target cluster.
func mustCreateClusterConfig(clusterID, caPath, host string) *rest.Config {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		log.WithError(err).Panic("failed to build kubernetes client config")
	}
	config.Host = host
	config.CAFile = caPath
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		return &addHeaderRoundTripper{
			headers: map[string][]string{XClusterIDHeader: {clusterID}},
			rt:      rt,
		}
	}
	return config
}

// addHeaderRoundTripper implements the http.RoundTripper interface and inserts the headers in headers field
// into the request made with an http.Client that uses this RoundTripper
type addHeaderRoundTripper struct {
	headers map[string][]string
	rt      http.RoundTripper
}

func (ha *addHeaderRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r2 := new(http.Request)
	*r2 = *r

	// To set extra headers, we must make a copy of the Request so
	// that we don't modify the Request we were given. This is required by the
	// specification of http.RoundTripper.
	//
	// Since we are going to modify only req.Header here, we only need a deep copy
	// of req.Header.
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}

	for key, values := range ha.headers {
		r2.Header[key] = values
	}

	return ha.rt.RoundTrip(r2)
}
