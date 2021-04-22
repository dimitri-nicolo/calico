// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package k8s

import (
	"context"
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

	"github.com/tigera/apiserver/pkg/client/clientset_generated/clientset"
	clientv3 "github.com/tigera/apiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

// Unexported type to avoid context key collisions.
type key int

const (
	clientSetKeyUser key = iota
	clientSetKeyApp
)

func GetClientSetApplicationFromContext(cxt context.Context) ClientSet {
	return cxt.Value(clientSetKeyApp).(ClientSet)
}

func GetClientSetUserFromContext(cxt context.Context) ClientSet {
	return cxt.Value(clientSetKeyUser).(ClientSet)
}

const (
	// Default cluster name for standalone and management cluster.
	DefaultCluster   = "cluster"
	XClusterIDHeader = "x-cluster-id"
)

type k8sInterface kubernetes.Interface
type calicoInterface clientv3.ProjectcalicoV3Interface

// ClientSet is a combined Calico/Kubernetes client set interface.
type ClientSet interface {
	k8sInterface
	calicoInterface
}

// ClientSetHandlers provides handler interfaces used to update the context with a Kubernetes ClientSet.
//
// Use the GetClientSet*FromContext helper functions to extract the client set from the context.
//
// The client set can either be tied to the application, or to the user (including any impersonation headers present
// in the original request).
type ClientSetHandlers interface {
	AddClientSetForApplication(handlerFunc http.Handler) http.HandlerFunc
	AddClientSetForUser(handlerFunc http.Handler) http.HandlerFunc
}

// clientSetHandlers is an implementation of the ClientSetHandlers interface.
//
// The clients added to the context are short lived - just used for the single request. There is no caching of these
// clients.
type clientSetHandlers struct {
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

// NewClientSetHandlers creates an implementation of the ClientSetHandlers.
func NewClientSetHandlers(multiClusterForwardingCA, multiClusterForwardingEndpoint string) ClientSetHandlers {
	return &clientSetHandlers{
		baseRestConfig:                 MustGetConfig(),
		multiClusterForwardingCA:       multiClusterForwardingCA,
		multiClusterForwardingEndpoint: multiClusterForwardingEndpoint,
	}
}

func (f *clientSetHandlers) AddClientSetForApplication(handlerFunc http.Handler) http.HandlerFunc {
	return f.addClientSet(handlerFunc, false, clientSetKeyApp)
}

func (f *clientSetHandlers) AddClientSetForUser(handlerFunc http.Handler) http.HandlerFunc {
	return f.addClientSet(handlerFunc, true, clientSetKeyUser)
}

func (f *clientSetHandlers) addClientSet(handlerFunc http.Handler, asUser bool, key key) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if cs, err := f.getClientSetForRequest(req, asUser); err != nil {
			log.WithError(err).Debug("Failed to create clientset")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			// Update the context to include the clientset and then chain to the next handler.
			cxt := context.WithValue(req.Context(), key, cs)
			req = req.WithContext(cxt)
			handlerFunc.ServeHTTP(w, req)
		}
	}
}

// getClientSetForRequest creates a new cluster-aware ClientSet and optionally for the user of the original request.
func (f *clientSetHandlers) getClientSetForRequest(origReq *http.Request, asUser bool) (ClientSet, error) {
	headers, err := f.headers(origReq, asUser)
	if err != nil {
		return nil, err
	}

	k8sConfig := f.restConfig(headers, asUser)
	calicoConfig := rest.CopyConfig(k8sConfig)

	calicoCli, err := clientset.NewForConfig(calicoConfig)
	if err != nil {
		return nil, err
	}

	k8sCli, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, err
	}

	return &clientSet{
		calicoInterface: calicoCli.ProjectcalicoV3(),
		k8sInterface:    k8sCli,
	}, nil
}

func (f *clientSetHandlers) headers(origReq *http.Request, asUser bool) (map[string][]string, error) {
	// Extract the cluster ID from the header and the user from the context.  The user is required.
	headers := map[string][]string{}

	// Copy the cluster ID if not the default cluster. We later check for non-existence of this key for indication
	// that it is the default cluster.
	if clusterID := origReq.Header.Get(XClusterIDHeader); clusterID != "" && clusterID != DefaultCluster {
		headers[XClusterIDHeader] = []string{clusterID}
	}

	if asUser {
		// Copy the authorization header.  We should have this in the request.
		if auth := origReq.Header.Values("Authorization"); len(auth) == 0 {
			return nil, errors.New("missing Authorization header in request")
		} else {
			headers["Authorization"] = auth
		}

		// Copy the impersonation info headers.  If the actual user is not allowed to impersonate then requests by this
		// client should fail.
		if user := origReq.Header.Values(transport.ImpersonateUserHeader); len(user) != 0 {
			headers[transport.ImpersonateUserHeader] = user
		}
		if group := origReq.Header.Values(transport.ImpersonateGroupHeader); len(group) != 0 {
			headers[transport.ImpersonateGroupHeader] = group
		}
		if extra := origReq.Header.Values(transport.ImpersonateUserExtraHeaderPrefix); len(extra) != 0 {
			headers[transport.ImpersonateUserExtraHeaderPrefix] = extra
		}
	}

	return headers, nil
}

// Create the rest.Config for the cluster/user.
func (f *clientSetHandlers) restConfig(headers map[string][]string, asUser bool) *rest.Config {
	restConfig := f.copyRESTConfig()

	if asUser {
		// Remove and bearer/user/impersonation info since we will use the appropriate headers from the original
		// request.
		restConfig.BearerToken = ""
		restConfig.Username = ""
		restConfig.Impersonate = rest.ImpersonationConfig{}
	}

	// Wrap to add the supplied headers if any.
	if len(headers) > 0 {
		if _, ok := headers[XClusterIDHeader]; ok {
			// If the cluster ID is present then update the host and cert for inter cluster forwarding.
			restConfig.Host = f.multiClusterForwardingEndpoint
			restConfig.CAFile = f.multiClusterForwardingCA
		}

		restConfig.Wrap(func(rt http.RoundTripper) http.RoundTripper {
			return &addHeaderRoundTripper{
				headers: headers,
				rt:      rt,
			}
		})
	}

	return restConfig
}

// copyRESTConfig returns a copy of the base rest config.
func (f *clientSetHandlers) copyRESTConfig() *rest.Config {
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
