package util

import (
	"net/http"

	"k8s.io/client-go/rest"

	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
)

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

// ManagedClusterClient returns a function that takes managed cluster name as input, then creates and returns calico client for that cluster.
func ManagedClusterClient(config *rest.Config, multiClusterForwardingEndpoint, multiClusterForwardingCA string) func(string) (calicoclient.Interface, error) {
	return func(clusterName string) (calicoclient.Interface, error) {
		config.Host = multiClusterForwardingEndpoint
		config.CAFile = multiClusterForwardingCA
		config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			return &addHeaderRoundTripper{
				headers: map[string][]string{"x-cluster-id": {clusterName}},
				rt:      rt,
			}
		}
		return calicoclient.NewForConfig(config)
	}
}
