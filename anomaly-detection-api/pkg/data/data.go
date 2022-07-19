package data

// LogTypeMetadata stores the request and response body for the
// /clusters/{cluser_name}/{log_type}/metadata endpoint, containing
// metadata processing information of the different log types for each
// cluster
type LogTypeMetadata struct {
	// LastUpdated is the decimaled unix timestamp for the most recent
	// time the cluster's log type logs were downloaded for training
	LastUpdated string `json:"last_updated"`
}
