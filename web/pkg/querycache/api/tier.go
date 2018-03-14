package api

type Tier interface {
	GetName() string
	GetResource() Resource
	GetOrderedPolicies() []Policy
}
