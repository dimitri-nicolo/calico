package api

type Policy interface {
	GetResource() Resource
	GetTier() string
	GetEndpointCounts() EndpointCounts
	GetRuleEndpointCounts() Rule
	IsUnmatched() bool
}

type PolicyCounts struct {
	NumGlobalNetworkPolicies int
	NumNetworkPolicies       int
}

type Rule struct {
	Ingress []RuleDirection
	Egress  []RuleDirection
}

type RuleDirection struct {
	Source      RuleEntity
	Destination RuleEntity
}

type RuleEntity struct {
	Selector    EndpointCounts
	NotSelector EndpointCounts
}
