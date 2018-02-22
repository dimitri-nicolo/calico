package api

type Endpoint interface {
	GetResource() Resource
	GetPolicyCounts() PolicyCounts
}

type EndpointCounts struct {
	NumWorkloadEndpoints int
	NumHostEndpoints     int
}
