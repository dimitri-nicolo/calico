package mock

import (
	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"github.com/tigera/intrusion-detection/controller/pkg/db"
)

type Call struct {
	Method  string
	GNS     *v3.GlobalNetworkSet
	Name    string
	Set     db.IPSetSpec
	Version *int64
}
