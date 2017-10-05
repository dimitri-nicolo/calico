package calico

import "k8s.io/apiserver/pkg/registry/generic"

const (
	PolicyResource string = "policy"
	TierResource   string = "tier"
)

type Options struct {
	RESTOptions generic.RESTOptions
}
