// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package rawquery

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"

	"github.com/tigera/es-proxy/pkg/middleware"
	"github.com/tigera/lma/pkg/httputils"
)

var (
	searchURLPattern *regexp.Regexp = regexp.MustCompile(`^/(.*)/_search$`)
)

// RawQueryHandle validates raw Elastic requests sent from Manager.
// It only accepts Elastic search requests in the following definition:
//   1. HTTP method: POST.
//   2. HTTP url: /<index>/_search.
//
// In Manager, the following pages are still sending raw Elastic search requests.
//   1. Dashboard page: total alerts count <= Calico Enterprise v3.12.
//   2. Alert List page: fetch security events <= Calico Enterprise v3.12.
//   3. Alert List page: fetch Kibana index pattern for logs.
//   4. Timeline page: fetch audit logs.
func RawQueryHandler(client *elastic.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// parse raw elastic request.
		index, body, err := parseQueryRequest(w, r)
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		// proxy search request to elastic.
		resp, err := client.Search().
			Index(index).
			Source(body).
			Do(r.Context())
		if err != nil {
			httputils.EncodeError(w, err)
			return
		}

		httputils.Encode(w, resp)
	})
}

func parseQueryRequest(w http.ResponseWriter, r *http.Request) (string, json.RawMessage, error) {
	if r.Method != http.MethodPost {
		log.WithError(middleware.ErrInvalidMethod).Errorf("Invalid http method %s for _search.", r.Method)
		return "", nil, &httputils.HttpStatusError{
			Status: http.StatusMethodNotAllowed,
			Msg:    middleware.ErrInvalidMethod.Error(),
			Err:    middleware.ErrInvalidMethod,
		}
	}

	match := searchURLPattern.FindStringSubmatch(r.URL.Path)
	if match == nil || len(match) != 2 {
		log.WithError(middleware.ErrParseRequest).Errorf("Invalid http url path %s for _search.", r.URL.Path)
		return "", nil, &httputils.HttpStatusError{
			Status: http.StatusBadRequest,
			Msg:    middleware.ErrParseRequest.Error(),
			Err:    middleware.ErrParseRequest,
		}
	}
	index := match[1]

	var body json.RawMessage
	if err := httputils.Decode(w, r, &body); err != nil {
		var mr *httputils.HttpStatusError
		if errors.As(err, &mr) {
			log.WithError(mr.Err).Info(mr.Msg)
			return "", nil, mr
		} else {
			log.WithError(mr.Err).Info("Error decoding _search body.")
			return "", nil, &httputils.HttpStatusError{
				Status: http.StatusBadRequest,
				Msg:    err.Error(),
				Err:    err,
			}
		}
	}

	return index, body, nil
}
