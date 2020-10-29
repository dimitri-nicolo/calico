package middleware

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	actionAllow             = "allow"
	actionDeny              = "deny"
	actionUnknown           = "unknown"
	flowTypeNetwork         = "net"
	flowTypeNetworkSet      = "ns"
	flowTypeWep             = "wep"
	flowTypeHep             = "hep"
	operatorEquals          = "="
	operatorNotEquals       = "!="
	policyPreviewVerbCreate = "create"
	policyPreviewVerbUpdate = "update"
	policyPreviewVerbDelete = "delete"
)

var (
	errInvalidMethod            = errors.New("Invalid http method")
	errParseRequest             = errors.New("Error parsing request parameters")
	errInvalidAction            = errors.New("Invalid action specified")
	errInvalidFlowType          = errors.New("Invalid flow type specified")
	errInvalidLabelSelector     = errors.New("Invalid label selector specified")
	errGeneric                  = errors.New("Something went wrong")
	errInvalidPolicyPreview     = errors.New("Invalid policy preview specified")
	errPreviewResourceExtraData = errors.New("Invalid policy preview specified - resource has unexpected data")
	errInvalidActionUnprotected = errors.New("Action deny and unprotected true is an invalid combination")
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

func validateActionsAndUnprotected(actions []string, unprotected bool) bool {
	if unprotected {
		for _, action := range actions {
			switch action {
			case actionDeny:
				//unprotected true and action deny cannot be both set
				return false
			default:
				continue
			}
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

func getPolicyPreviews(previews []string) ([]PolicyPreview, error) {
	if len(previews) == 0 {
		return nil, nil
	}
	var policyPreviews []PolicyPreview

	// Decode the policy preview JSON data. We should fail if there are unhandled fields in the request. Validation of
	// the actual data is done within PIP as part of the xrefcache population.
	for _, preview := range previews {
		var policyPreview PolicyPreview
		decoder := json.NewDecoder(strings.NewReader(preview))
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&policyPreview)
		if err != nil {
			return nil, err
		}
		if decoder.More() {
			return nil, errPreviewResourceExtraData
		}
		policyPreviews = append(policyPreviews, policyPreview)
	}
	return policyPreviews, nil
}

func validatePolicyPreviews(policyPreviews []PolicyPreview) bool {
	if policyPreviews == nil {
		return true
	}

	for _, policyPreview := range policyPreviews {
		if policyPreview.Verb == "" || policyPreview.NetworkPolicy == nil {
			return false
		}
		switch policyPreview.Verb {
		case policyPreviewVerbCreate:
			break
		case policyPreviewVerbUpdate:
			break
		case policyPreviewVerbDelete:
			break
		default:
			return false
		}
	}
	return true
}
