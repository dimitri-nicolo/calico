package middleware

import (
	elastic "github.com/olivere/elastic/v7"
)

// UnprotectedQuery returns an elastic nested query
//	"nested": {
//	  "path": "policies",
//	  "query": {
//		"wildcard": {
//		  "policies.all_policies": {
//			"_name": "value",
//			"wildcard": "*|__PROFILE__|__PROFILE__.kns.*|allow*"
//		  }
//		}
//	  }
//  }
func UnprotectedQuery() *elastic.NestedQuery {
	// wildcard is an area for improvement. Based on doc "Avoid beginning patterns with * or ?.
	// This can increase the iterations needed to find matching terms and slow search performance"
	wildcardQuery := elastic.NewWildcardQuery("policies.all_policies", "*|__PROFILE__|__PROFILE__.kns.*|allow*")
	return elastic.NewNestedQuery("policies", wildcardQuery)
}
