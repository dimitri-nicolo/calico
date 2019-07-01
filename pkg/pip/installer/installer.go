package installer

import (
	"github.com/tigera/compliance/pkg/datastore"

	"github.com/tigera/es-proxy/pkg/handler"
	"github.com/tigera/es-proxy/pkg/middleware"
	"github.com/tigera/es-proxy/pkg/mutator"
	"github.com/tigera/es-proxy/pkg/pip"
)

// InstallPolicyImpactResponseHook connects up the pip response mutator
// The response mutator uses the policyActions and returned flow data
// to call the primary PIP calculation and then replace the flow data
// destined for the client with the modified flow data returned from the
// PIP calculation
func InstallPolicyImpactReponseHook(proxy *handler.Proxy, client datastore.ClientSet) {
	p := pip.New(client)
	piphook := mutator.NewPIPResponseHook(p)
	proxy.AddResponseModifier(piphook.ModifyResponse)
}
