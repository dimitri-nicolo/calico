package middleware

import (
	log "github.com/sirupsen/logrus"
	"net/url"
	"strconv"
	"strings"
)

const (
	actionAllow   = "allow"
	actionDeny    = "deny"
	actionUnknown = "unknown"
)

func extractLimitParam(url url.Values) (int32, error) {
	var limit int32
	limitParam := url.Get("limit")
	if limitParam == "" || limitParam == "0" {
		limit = 1000
	} else {
		parsedLimit, err := strconv.ParseInt(limitParam, 10, 32)
		if err != nil || parsedLimit < 0 {
			log.WithError(err).Info("Error parsing limit parameter")
			return 0, errParseRequest
		}
		limit = int32(parsedLimit)
	}
	return limit, nil
}

func lowerCaseActions(actions []string) []string {
	for i, action := range actions {
		actions[i] = strings.ToLower(action)
	}
	return actions
}

func validateActions(actions []string) bool {
	for _, action := range actions {
		switch action {
		case actionAllow:
			continue
		case actionDeny:
			continue
		case actionUnknown:
			continue
		default:
			return false
		}
	}
	return true
}
