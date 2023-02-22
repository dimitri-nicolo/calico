package v1

// ProcessParams define querying parameters to retrieve process data from flow logs.
type ProcessParams struct {
	QueryParams        `json:",inline" validate:"required"`
	LogSelectionParams `json:",inline"`
}

type ProcessInfo struct {
	// Name of the process.
	Name string `json:"name" validate:"required"`

	// Endpoint that executed the process.
	Endpoint string `json:"endpoint" validate:"required"`

	// Count the number of instances of this process.
	Count int `json:"count" validate:"required"`
}
