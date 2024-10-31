// Copyright (c) 2024 Tigera, Inc. All rights reserved
package utils

import (
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"

	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
)

const (
	// ClusterLoggingID is the field used to identify the cluster logging resource.
	ManagementLoggingID = "management"
)

// GetLogClusterID returns the field used to identify the cluster resource for logging purposes.
func GetLogClusterID(id string) string {
	field := ManagementLoggingID
	if field != "cluster" {
		// If the cluster ID is not "cluster", use the managed cluster ID.
		field = id
	}
	return field
}

// GenerateRecommendationName returns a policy name with tier prefix and 5 char hash suffix. If there is an
// error in generating the policy name, it returns an empty string.
func GenerateRecommendationName(tier, name string, suffixGenerator func() string) string {
	policy, err := getRFC1123PolicyName(tier, name, suffixGenerator())
	if err != nil {
		log.WithError(err).Errorf("Failed to generate policy name for tier %s and name %s", tier, name)
	}

	return policy
}

// SuffixGenerator returns a random 5 char string, typically used as a suffix to a name in
// Kubernetes.
func SuffixGenerator() string {
	return testutils.RandStringRunes(5)
}

// getRFC1123PolicyName returns an RFC 1123 compliant policy name.
// Returns a policy name with the following format: <TIER>.<NAME>-<SUFFIX>, if the length is valid.
// Otherwise, it cuts the <TIER>.<NAME> down to size for validity, and returns the adapted policy
// name followed by the suffix.
// The tier name and the name are assumed to be valid RFC 1123 compliant names. The suffix is a
// random 5 char string.
func getRFC1123PolicyName(tier, name, suffix string) (string, error) {
	if tier == "" || name == "" {
		return "", fmt.Errorf("either tier name '%s' or policy name '%s' is empty", tier, name)
	}

	max := k8svalidation.DNS1123LabelMaxLength - (len(suffix) + 1)
	if len(tier)+2 > max {
		return "", fmt.Errorf("tier name %s is too long to be used in a policy name", tier)
	}

	policy := fmt.Sprintf("%s.%s", tier, name)
	if len(policy) > max {
		// Truncate policy name to max length
		policy = policy[:max]
	}

	return fmt.Sprintf("%s-%s", policy, suffix), nil
}

// ExtractMessages extracts the messages from a log string.
func ExtractMessages(logString string) []string {
	// Split the log string into individual log entries
	logs := strings.Split(logString, "\n")

	// Regular expression to extract the message from each log entry
	re := regexp.MustCompile(`msg="([^"]+)"`)

	var messages []string

	// Iterate over each log entry and extract the message
	for _, logEntry := range logs {
		// FindStringSubmatch returns an array where index 0 is the full match,
		// and index 1 is the first captured group (our message)
		match := re.FindStringSubmatch(logEntry)
		if len(match) >= 2 {
			messages = append(messages, match[1])
		}
	}

	return messages
}
