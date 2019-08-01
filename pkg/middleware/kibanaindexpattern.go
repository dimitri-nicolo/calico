package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	log "github.com/sirupsen/logrus"
)

// The handler returned by this will add a ResourceAttribute to the context
// of the request based on the content of the kibana query index-pattern
// (query.bool.filter.match.index-pattern.title)
func KibanaIndexPatern(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		name, err := getResourceNameFromKibanaIndexPatern(req)
		if err != nil {
			log.WithError(err).Debugf("Unable to extract kibana index patern as resource")
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, req.WithContext(NewContextWithReviewResource(req.Context(), getResourceAttributes(name))))
	})
}

// getResourceNameFromKibanaIndexPatern parses the query.bool.filter.match.index-pattern.title
// from a kibana query request body and returns the RBAC resource
func getResourceNameFromKibanaIndexPatern(req *http.Request) (string, error) {

	// Read the body data
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.WithError(err).Debug("Error reading request body")
		return "", err
	}

	//  reset the request body
	req.Body = ioutil.NopCloser(bytes.NewBuffer(b))

	// unmarshal the json
	var k kibanaReq
	err = json.Unmarshal(b, &k)
	if err != nil {
		log.WithError(err).WithField("body", string(b[:])).Debug("JSON parse error")
		return "", err
	}

	// extract the index pattern title
	title := k.Query.Bool.Filter[0].Match.IndexPatternTitle

	re := regexp.MustCompile(`([_a-z]*)`)

	titleMatch := re.FindStringSubmatch(title)
	if len(titleMatch) != 2 {
		return "", fmt.Errorf("Invalid index pattern in title, '%s' had %d matches", title, len(titleMatch))
	}

	resource, ok := queryToResource(titleMatch[0])
	if !ok {
		return "", fmt.Errorf("Invalid resource '%s' in kibana index-pattern", title)
	}
	log.WithField("title", title).WithField("resource", resource).Info("kibana index-pattern")
	return resource, nil
}

// kibanaReq and kibanaReqMatch are for parsing a json doc formatted like this:
// {
//     "query": {
//         "bool": {
//             "filter": [
//                 {
//                     "match": {
//                         "index-pattern.title": "tigera_secure_ee_flows"
//                     }
//                 }
//             ]
//         }
//     }
// }

type kibanaReq struct {
	Query struct {
		Bool struct {
			Filter []kibanaReqMatch `json:"filter"`
		} `json:"bool"`
	} `json:"query"`
}

type kibanaReqMatch struct {
	Match struct {
		IndexPatternTitle string `json:"index-pattern.title"`
	} `json:"match"`
}
