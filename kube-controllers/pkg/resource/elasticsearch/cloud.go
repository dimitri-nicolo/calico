// Copyright (c) 2021 Tigera, Inc. All rights reserved.

// +build tesla

package elasticsearch

// CalculateTigeraElasticsearchHash for the Cloud/Tesla variant simply returns a string and a nil error
// since the cluster does not contain an Elasticsearch CR.
func (r *restClient) CalculateTigeraElasticsearchHash() (string, error) {
	// Returning a non-empty string ensures that we perform a one time creation of Elasticsearch roles.
	return "externalElasticsearch", nil
}
