package api

type Endpoint interface {
	GetResource() Resource
	GetNode() string
	GetPolicyCounts() PolicyCounts
	IsProtected() bool
	IsLabelled() bool
}

type EndpointCounts struct {
	NumWorkloadEndpoints int
	NumHostEndpoints     int
}
