// Copyright (c) 2021 Tigera, Inc. All rights reserved.
package servicegraph

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LabelSelectors are calico expressions to match labels
type LabelSelectors []string

func AppendLabelSelectors(source LabelSelectors, selector *metav1.LabelSelector) LabelSelectors {
	if selector == nil {
		return source
	}

	for k, v := range selector.MatchLabels {
		source = append(source, fmt.Sprintf("%s in {\"%s\"}", k, v))
	}

	for _, sel := range selector.MatchExpressions {
		var values []string
		for _, v := range sel.Values {
			values = append(values, fmt.Sprintf("\"%s\"", v))
		}

		switch sel.Operator {
		case metav1.LabelSelectorOpIn:
			source = append(source, fmt.Sprintf("%s in {%s}", sel.Key, strings.Join(values, ",")))
		case metav1.LabelSelectorOpNotIn:
			source = append(source, fmt.Sprintf("%s not in {%s}", sel.Key, strings.Join(values, ",")))
		case metav1.LabelSelectorOpExists:
			source = append(source, fmt.Sprintf("has(%s)", sel.Key))
		case metav1.LabelSelectorOpDoesNotExist:
			source = append(source, fmt.Sprintf("!has(%s)", sel.Key))
		}
	}

	return source
}

func AppendLabels(source LabelSelectors, labels map[string]string) LabelSelectors {
	if labels == nil {
		return source
	}

	for k, v := range labels {
		source = append(source, fmt.Sprintf("%s == \"%s\"", k, v))
	}

	return source
}
