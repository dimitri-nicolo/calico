package installer

import (
	"github.com/tigera/compliance/pkg/datastore"

	"github.com/tigera/es-proxy/pkg/handler"
	"github.com/tigera/es-proxy/pkg/mutator"
	"github.com/tigera/es-proxy/pkg/pip"
	"github.com/tigera/es-proxy/pkg/pip/policycalc"
)

// InstallPolicyImpactResponseHook connects up the pip response mutator
// The response mutator uses the policyActions and returned flow data
// to call the primary PIP calculation and then replace the flow data
// destined for the client with the modified flow data returned from the
// PIP calculation
func InstallPolicyImpactReponseHook(proxy *handler.Proxy, policyCalcConfig *policycalc.Config, client datastore.ClientSet) {
	p := pip.New(policyCalcConfig, client)
	piphook := mutator.NewPIPResponseHook(p)
	proxy.AddResponseModifier(piphook.ModifyResponse)
}
