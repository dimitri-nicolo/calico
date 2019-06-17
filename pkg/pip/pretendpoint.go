package pip

import (
	"strings"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/tigera/es-proxy/pkg/pip/flow"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: support heps too
func EndpointsFromFlow(f flow.Flow) (*v3.WorkloadEndpoint, *v3.WorkloadEndpoint) {
	// TODO: return Nil for first val if source is not a k8s pod
	srcEp := &v3.WorkloadEndpoint{
		ObjectMeta: v1.ObjectMeta{
			Namespace: f.Src_NS,
			Labels:    f.Src_labels,
		},
		Spec: v3.WorkloadEndpointSpec{
			Pod: strings.TrimSuffix(f.Src_name, "-*"),
		},
	}
	srcEp.Labels["projectcalico/namespace"] = f.Src_NS

	destEp := &v3.WorkloadEndpoint{
		ObjectMeta: v1.ObjectMeta{
			Namespace: f.Dest_NS,
			Labels:    f.Dest_labels,
		},
		Spec: v3.WorkloadEndpointSpec{
			Pod: strings.TrimSuffix(f.Dest_name, "-*"),
		},
	}
	srcEp.Labels["projectcalico/namespace"] = f.Dest_NS

	return srcEp, destEp
}
