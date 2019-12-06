package middleware

import (
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	actionAllow             = "allow"
	actionDeny              = "deny"
	actionUnknown           = "unknown"
	flowTypeNetwork         = "network"
	flowTypeNetworkSet      = "networkset"
	flowTypeWep             = "wep"
	flowTypeHep             = "hep"
	operatorEquals         = "="
	operatorNotEquals      = "!="
)

var (
	errInvalidMethod        = errors.New("Invalid http method")
	errParseRequest         = errors.New("Error parsing request parameters")
	errInvalidAction        = errors.New("Invalid action specified")
	errInvalidFlowType      = errors.New("Invalid flow type specified")
	errInvalidLabelSelector = errors.New("Invalid label selector specified")
	errGeneric              = errors.New("Something went wrong")
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

func lowerCaseParams(params []string) []string {
	for i, param := range params {
		params[i] = strings.ToLower(param)
	}
	return params
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

func getLabelSelectors(labels []string) ([]LabelSelector, error) {
	if len(labels) == 0 {
		return nil, nil
	}
	labelSelectors := make([]LabelSelector, len(labels))
	for i, label := range labels {
		labelSelector := LabelSelector{}
		err := json.Unmarshal([]byte(label), &labelSelector)
		if err != nil {
			return nil, err
		}
		labelSelectors[i] = labelSelector
	}

	return labelSelectors, nil
}

func validateFlowTypes(flowTypes []string) bool {
	for _, flowType := range flowTypes {
		switch flowType {
		case flowTypeNetwork:
			continue
		case flowTypeNetworkSet:
			continue
		case flowTypeWep:
			continue
		case flowTypeHep:
			continue
		default:
			return false
		}
	}
	return true
}

func convertFlowTypes(flowTypes []string) []string {
	var convertedFlowTypes []string
	for _, flowType := range flowTypes {
		var convertedFlowType string
		switch flowType {
		case flowTypeNetwork:
			convertedFlowType = "net"
		case flowTypeNetworkSet:
			convertedFlowType = "ns"
		default:
			convertedFlowType = flowType
		}
		convertedFlowTypes = append(convertedFlowTypes, convertedFlowType)
	}
	return convertedFlowTypes
}

func validateLabelSelector(labelSelectors []LabelSelector) bool {
	// validate operator/match type
	for _, labelSelector := range labelSelectors {
		// make sure all required fields are present
		if labelSelector.Key == "" || labelSelector.Operator == "" || len(labelSelector.Values) == 0 {
			return false
		}
		switch labelSelector.Operator {
		case operatorEquals:
			continue
		case operatorNotEquals:
			continue
		default:
			return false
		}
	}
	return true
}

func parseAndValidateTime(timeString string) (time.Time, error) {
	if timeString == "" {
		return time.Time{}, nil
	}
	parsedTime, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		return time.Time{}, err
	}
	return parsedTime, nil
}
