package pip

import (
	"encoding/json"

	"github.com/tigera/compliance/pkg/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceChange contains a single resource update that we are previewing.
type ResourceChange struct {
	Action   string             `json:"action"`
	Resource resources.Resource `json:"resource"`
}

// resourceChangeTrial is used to temporarily unmarshal the ResourceChange so that we can extract the TypeMeta from
// the resource definition.
type resourceChangeTrial struct {
	Resource metav1.TypeMeta `json:"resource"`
}

// Defined an alias for the ResourceChange so that we can json unmarshal it from the ResourceChange.UnmarshalJSON
// without causing recursion (since aliased types do not inherit methods).
type AliasedResourceChange *ResourceChange

// UnmarshalJSON allows unmarshalling of a ResourceChange from JSON bytes. This is required because the Resource
// field is an interface, and so it needs to be set with a concrete type before it can be unmarshalled.
func (c *ResourceChange) UnmarshalJSON(b []byte) error {
	// Unmarshal into the "trial" struct that allows us to easily extract the TypeMeta of the resource.
	var r resourceChangeTrial
	if err := json.Unmarshal(b, &r); err != nil {
		return err
	}
	c.Resource = resources.NewResource(r.Resource)
	return json.Unmarshal(b, AliasedResourceChange(c))
}
