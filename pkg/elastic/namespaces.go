// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package elastic

import (
	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
)

// GetNamespacesQuery returns the query to enumerate endpoints with the specified namespaces
func GetNamespacesQuery(namespaces []string) elastic.Query {
	log.Debugf("Construct namespaces query for: %v", namespaces)
	return elastic.NewBoolQuery().Should(
		elastic.NewTermsQueryFromStrings("source_namespace", namespaces...),
		elastic.NewTermsQueryFromStrings("dest_namespace", namespaces...),
	)
}
