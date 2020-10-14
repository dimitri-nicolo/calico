package datastore

import (
	"net/http"
	"os"
	"sync"

	"github.com/tigera/compliance/pkg/config"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/apiserver/pkg/authentication"
	"github.com/tigera/apiserver/pkg/client/clientset_generated/clientset"
	v3 "github.com/tigera/apiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	"github.com/tigera/lma/pkg/auth"
	"github.com/tigera/lma/pkg/elastic"
)

const (
	// Default cluster name for standalone and management cluster.
	DefaultCluster   = "cluster"
	XClusterIDHeader = "x-cluster-id"
)

// Bundles clientMap behind convenience methods for multi cluster setups.
type RESTClientFactory interface {
	ClientSet(clusterID string) ClientSet
	K8sClient(clusterID string) kubernetes.Interface
	CalicoClient(clusterID string) v3.ProjectcalicoV3Interface
	ESClient(clusterID string) elastic.Client
	K8sAuth(clusterID string) auth.K8sAuthInterface
}

// Loads current cluster eagerly, loads external clusters lazily.
type RESTClientHolder struct {
	clientMap     map[string]clusterClients
	authenticator authentication.Authenticator
	sync.Mutex
	multiClusterForwardingCA       string
	multiClusterForwardingEndpoint string
}

// Groups clients per cluster
type clusterClients struct {
	calico  v3.ProjectcalicoV3Interface
	k8s     kubernetes.Interface
	es      elastic.Client
	k8sAuth auth.K8sAuthInterface
}

// Creates an instance of RESTClientFactory, which holds various clients.
// Loads current cluster eagerly, loads external clusters lazily.
func MustGetRESTClient(config *config.Config) RESTClientFactory {
	authenticator, err := authentication.New()
	if err != nil {
		log.WithError(err).Panic("Unable to create auth configuration")
	}

	if config.DexEnabled {

		opts := []auth.DexOption{
			auth.WithGroupsClaim(config.DexGroupsClaim),
			auth.WithJWKSURL(config.DexJWKSURL),
			auth.WithUsernamePrefix(config.DexUsernamePrefix),
			auth.WithGroupsPrefix(config.DexGroupsPrefix),
		}

		dex, err := auth.NewDexAuthenticator(
			config.DexIssuer,
			config.DexClientID,
			config.DexUsernameClaim,
			opts...)

		if err != nil {
			log.WithError(err).Panic("Unable to create dex authenticator")
		}
		authenticator = auth.NewAggregateAuthenticator(dex, authenticator)
	}

	k8sClient := MustGetKubernetesClient()
	return &RESTClientHolder{
		authenticator:                  authenticator,
		multiClusterForwardingCA:       config.MultiClusterForwardingCA,
		multiClusterForwardingEndpoint: config.MultiClusterForwardingEndpoint,
		clientMap: map[string]clusterClients{
			DefaultCluster: {
				MustGetCalicoClient(),
				k8sClient,
				elastic.MustGetElasticClient(),
				auth.NewK8sAuth(k8sClient, authenticator),
			},
		},
	}
}

// Return a ClientSet that lets you fetch Calico and Kubernetes resources from the datastore for a given cluster.
func (c *RESTClientHolder) ClientSet(clusterID string) ClientSet {
	c.Lock()
	defer c.Unlock()
	if clusterID == "" {
		clusterID = DefaultCluster
	} else if _, ok := c.clientMap[clusterID]; !ok {
		c.mustAddCluster(clusterID, c.multiClusterForwardingCA, c.multiClusterForwardingEndpoint)
	}
	return &clientSet{
		k8sInterface:    c.clientMap[clusterID].k8s,
		calicoInterface: c.clientMap[clusterID].calico,
	}
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

// Return a client that lets you fetch kubernetes objects from the datastore for a given cluster.
func (c *RESTClientHolder) K8sClient(clusterID string) kubernetes.Interface {
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
		return c.clientMap[DefaultCluster].k8sAuth
	}
	if _, ok := c.clientMap[clusterID]; !ok {
		c.mustAddCluster(clusterID, c.multiClusterForwardingCA, c.multiClusterForwardingEndpoint)
	}
	return c.clientMap[clusterID].k8sAuth
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
		k8sClient,
		mustCreateESClientForCluster(clusterID),
		auth.NewK8sAuth(k8sClient, c.authenticator),
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
