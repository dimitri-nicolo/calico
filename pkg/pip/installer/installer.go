package installer

import (
	"net/http"

	"github.com/tigera/es-proxy/pkg/pip/datastore"

	"github.com/tigera/es-proxy/pkg/middleware"
	"github.com/tigera/es-proxy/pkg/pip"

	"github.com/tigera/es-proxy/pkg/handler"
	"github.com/tigera/es-proxy/pkg/mutator"
)

// InstallPolicyImpactPreview hooks up both the pip response hook,
// and the pip middlware.
// The pip middleware extracts policyActions from the POST request
// body after es rbac checks but prior to the query to elastic search.
// It passes the policyActions to the policy impact mutator via the request
// context.
// The response mutor then uses the policyActions and returned flow data
// to call the primary PIP calculation and then replace the flow data
// destined for the client with the modified flow data returned from the
// PIP calculation
func InstallPolicyImpactPreview(listSrc datastore.ClientSet, proxy *handler.Proxy) http.Handler {
	//hook up the pip response modifier
	p := pip.New(listSrc)
	piphook := mutator.NewPIPResponseHook(p)
	proxy.AddResponseModifier(piphook.ModifyResponse)

	//return the policy impact handler
	return middleware.PolicyImpactParamsHandler(proxy)
}
