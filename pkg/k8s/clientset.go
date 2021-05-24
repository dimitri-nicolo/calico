// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package k8s

import (
	"errors"
	"net/http"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"

	"github.com/projectcalico/apiserver/pkg/client/clientset_generated/clientset"
	clientv3 "github.com/projectcalico/apiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

const (
	// Default cluster name for standalone and management cluster.
	DefaultCluster = "cluster"

	// The cluster ID to include in the x-headers of the modified HTTP client.
	XClusterIDHeader = "x-cluster-id"
)

type k8sInterface kubernetes.Interface
type calicoInterface clientv3.ProjectcalicoV3Interface

type ClientSetFactory interface {
	NewClientSetForUser(req *http.Request, cluster string) (ClientSet, error)
	NewClientSetForApplication(cluster string) (ClientSet, error)
}

// ClientSet is a combined Calico/Kubernetes client set interface.
type ClientSet interface {
	k8sInterface
	calicoInterface
}

// clientSetFactory is a factory for creating user-specific and cluster specific kubernetes/calico clientsets.
type clientSetFactory struct {
	sync.Mutex
	baseRestConfig                 *rest.Config
	multiClusterForwardingCA       string
	multiClusterForwardingEndpoint string
}

// The client set struct implementing the client set interface.
type clientSet struct {
	k8sInterface
	calicoInterface
}

// NewClientSetFactory creates an implementation of the ClientSetHandlers.
func NewClientSetFactory(multiClusterForwardingCA, multiClusterForwardingEndpoint string) ClientSetFactory {
	return &clientSetFactory{
		baseRestConfig:                 MustGetConfig(),
		multiClusterForwardingCA:       multiClusterForwardingCA,
		multiClusterForwardingEndpoint: multiClusterForwardingEndpoint,
	}
}

func (f *clientSetFactory) NewClientSetForApplication(clusterID string) (ClientSet, error) {
	return f.getClientSet(nil, clusterID)
}

func (f *clientSetFactory) NewClientSetForUser(req *http.Request, clusterID string) (ClientSet, error) {
	return f.getClientSet(req, clusterID)
}

func (f *clientSetFactory) getClientSet(req *http.Request, clusterID string) (ClientSet, error) {
	// Copy the rest config.
	restConfig := f.copyRESTConfig()

	// Determine which headers to override.
	headers := map[string][]string{}

	// If not the default cluster then add a cluster header.
	if clusterID != "" && clusterID != DefaultCluster {
		headers[XClusterIDHeader] = []string{clusterID}

		// In this case, update the host and cert for inter-cluster forwarding.
		restConfig.Host = f.multiClusterForwardingEndpoint
		restConfig.CAFile = f.multiClusterForwardingCA
	}

	// If the request has been specified then we are after a user-specific client set, so add the users bearer token
	// and impersonation info.
	if req != nil {
		// Copy the authorization header.  We should have this in the request.
		if auth := req.Header.Values("Authorization"); len(auth) == 0 {
			return nil, errors.New("missing Authorization header in request")
		} else {
			headers["Authorization"] = auth
		}

		// Copy the impersonation info headers.  If the actual user is not allowed to impersonate then requests by this
		// client should fail.
		if user := req.Header.Values(transport.ImpersonateUserHeader); len(user) != 0 {
			headers[transport.ImpersonateUserHeader] = user
		}
		if group := req.Header.Values(transport.ImpersonateGroupHeader); len(group) != 0 {
			headers[transport.ImpersonateGroupHeader] = group
		}
		if extra := req.Header.Values(transport.ImpersonateUserExtraHeaderPrefix); len(extra) != 0 {
			headers[transport.ImpersonateUserExtraHeaderPrefix] = extra
		}

		// Since we explicitly updating these headers, just remove authentication info from the rest config.
		restConfig.BearerToken = ""
		restConfig.Username = ""
		restConfig.Impersonate = rest.ImpersonationConfig{}
	}

	// Wrap to add the supplied headers if any.
	if len(headers) > 0 {
		restConfig.Wrap(func(rt http.RoundTripper) http.RoundTripper {
			return &addHeaderRoundTripper{
				headers: headers,
				rt:      rt,
			}
		})
	}

	calicoCli, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	k8sCli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &clientSet{
		calicoInterface: calicoCli.ProjectcalicoV3(),
		k8sInterface:    k8sCli,
	}, nil
}

// copyRESTConfig returns a copy of the base rest config.
func (f *clientSetFactory) copyRESTConfig() *rest.Config {
	f.Lock()
	defer f.Unlock()
	return rest.CopyConfig(f.baseRestConfig)
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

// MustGetConfig returns the rest Config for the local cluster.
func MustGetConfig() *rest.Config {
	kubeconfig := os.Getenv("KUBECONFIG")
	var config *rest.Config
	var err error
	if kubeconfig == "" {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			log.WithError(err).Panic("Error getting in-cluster config")
		}
	} else {
		// creates a config from supplied kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.WithError(err).Panic("Error processing kubeconfig file in environment variable KUBECONFIG")
		}
	}
	config.Timeout = 15 * time.Second
	return config
}
