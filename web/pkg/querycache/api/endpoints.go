package api

type Endpoint interface {
	GetResource() Resource
	GetNode() string
	GetPolicyCounts() PolicyCounts
}

type EndpointCounts struct {
	NumWorkloadEndpoints int
	NumHostEndpoints     int
}
