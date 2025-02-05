package v1

// ProcessParams define querying parameters to retrieve process data from flow logs.
type ProcessParams struct {
	QueryParams        `json:",inline" validate:"required"`
	LogSelectionParams `json:",inline"`
}

func (p ProcessParams) SetSortBy(sort []SearchRequestSortBy) {
	panic("implement me")
}

func (p ProcessParams) GetSortBy() []SearchRequestSortBy {
	return nil
}

type ProcessInfo struct {
	// Name of the process.
	Name string `json:"name" validate:"required"`

	// Endpoint that executed the process.
	Endpoint string `json:"endpoint" validate:"required"`

	// Count the number of instances of this process.
	// NOTE: The json tag is not consistent with others because it is
	// using the format expected by the UI.
	Count int `json:"instanceCount" validate:"required"`

	Cluster string `json:"cluster" validate:"required"`
}
