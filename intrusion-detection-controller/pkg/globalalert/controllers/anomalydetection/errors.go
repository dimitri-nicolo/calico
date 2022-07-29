// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package anomalydetection

type PodTemplateError struct{}

func (m *PodTemplateError) Error() string {
	return "PodTemplateError"
}
