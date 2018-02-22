package api

type Node interface {
	GetName() string
	GetResource() Resource
	GetEndpointCounts() EndpointCounts
}
