package middleware

import (
	"net/http"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/tigera/lma/pkg/auth"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// Default cluster name for standalone and management cluster.
	DefaultCluster    = "cluster"
	voltronServiceURL = "https://localhost:9443"
	XClusterIDHeader  = "x-cluster-id"
)

// Multi cluster requests work as follows:
// - A user, Jane, wants to use the manager UI to look at a managed cluster.
// - Jane has rbac permissions to use the manager UI inside the management cluster
// - Jane has Elasticsearch permissions (lma) inside the management cluster
// - Moreover, Jane has rbac permissions to list pods etc in the managed cluster
// - When she makes a call to the manager UI, her bearer token is attached to the header
// - Voltron receives this request and sends it to es-proxy (this artifact)
// - es-proxy performs authn & authz for the purpose of using the UI
// - es-proxy fetches the data from es
// - es-proxy then makes a rbac calls to see what jane can see in the managed cluster.
//
// This is how the RBAC calls are made to the other cluster:
// - We replace the host of the k8s-api server to the voltronServiceURL and add the XClusterIDHeader
// - Voltron will read this header and send it through a tunnel to Guardian in the managed cluster
// - On the other side of the tunnel Guardian impersonates es-proxy-server (sa=tigera-manager) to do
//     subject access reviews for Jane
// - The results are returned back to the RBACHelper. See flowauth.go to see how the cluster is taken
//     from the request context.
type MCMAuth interface {
	DefaultK8sAuth() auth.K8sAuthInterface
	K8sAuth(clusterID string) auth.K8sAuthInterface
}

// Return an object that can provide k8s interfaces for a given cluster id.
func NewMCMAuth(voltronCAPath string) MCMAuth {
	a := mcmAuth{
		clients:       map[string]auth.K8sAuthInterface{},
		voltronCAPath: voltronCAPath,
	}
	// Initialize the default cluster.
	a.K8sAuth(DefaultCluster)
	return &a
}

type mcmAuth struct {
	clients       map[string]auth.K8sAuthInterface
	voltronCAPath string
	sync.RWMutex
}

// Return a client that lets you check authn and authz for the default cluster.
func (c *mcmAuth) DefaultK8sAuth() auth.K8sAuthInterface {
	c.RLock()
	defer c.RUnlock()
	return c.clients[DefaultCluster]
}

// Return a client that lets you check authn and authz for a given cluster.
// Writes to the mcmAuth map happen occasionally over its lifespan, while reads occur more than once per request. We
// minimize full locks as much as possible, such that the best possible performance is achieved, while supporting
// concurrent requests.
func (c *mcmAuth) K8sAuth(clusterID string) auth.K8sAuthInterface {
	if clusterID == "" {
		return c.DefaultK8sAuth()
	}

	c.RLock()
	if k8sauth, ok := c.clients[clusterID]; ok {
		c.RUnlock()
		return k8sauth
	}
	c.RUnlock()
	k8sauth := createClusterConfig(clusterID, c.voltronCAPath)

	c.Lock()
	c.clients[clusterID] = k8sauth
	c.Unlock()
	return k8sauth
}

// Adds a new k8s client to the map and also returns it.
func createClusterConfig(clusterID, voltronCAPath string) auth.K8sAuthInterface {
	cfg := mustCreateClusterConfig(clusterID, voltronCAPath)
	k8sClient := k8s.NewForConfigOrDie(cfg)
	return auth.NewK8sAuth(k8sClient, cfg)
}

// Create a config that routes requests through Voltron to a target cluster.
func mustCreateClusterConfig(clusterID, voltronCAPath string) *rest.Config {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		log.WithError(err).Panic("failed to build kubernetes client config")
	}
	if clusterID != DefaultCluster {
		config.Host = voltronServiceURL
		config.CAFile = voltronCAPath
		config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			return &addHeaderRoundTripper{
				headers: map[string][]string{XClusterIDHeader: {clusterID}},
				rt:      rt,
			}
		}
	}
	return config
}

// AddHeaderRoundTripper implements the http.RoundTripper interface and inserts the headers in headers field
// into the request made with an http.Client that uses this RoundTripper
type addHeaderRoundTripper struct {
	headers map[string][]string
	rt      http.RoundTripper
}

// Adds header such that Voltron will redirect the request to a k8s api server in a different cluster.
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
		r2.Header[k] = append(make([]string, 0, len(s)), s...)
	}

	for key, values := range ha.headers {
		r2.Header[key] = values
	}
	return ha.rt.RoundTrip(r2)
}
