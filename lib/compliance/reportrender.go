// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compliance

import (
	"bytes"
	"fmt"
	"reflect"
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

// Exposed to be used by UT code.
var ResourceIdSample = api.ResourceID{
	TypeMeta: metav1.TypeMeta{
		Kind:       "sample-kind",
		APIVersion: "projectcalico.org/v3",
	},
	Name:      "sample-res",
	Namespace: "sample-ns",
}
var EndpointSample = api.EndpointsReportEndpoint{
	ID:               ResourceIdSample,
	IngressProtected: false,
	EgressProtected:  true,
	EnvoyEnabled:     false,
	AppliedPolicies:  []api.ResourceID{ResourceIdSample, ResourceIdSample},
	Services:         []api.ResourceID{ResourceIdSample, ResourceIdSample},
}

// ReportDataSample is used by ReportTemplate validator.
var ReportDataSample = api.ReportData{
	StartTime: metav1.Unix(1554076800, 0),
	EndTime:   metav1.Unix(1554112800, 0),
	ReportSpec: api.ReportSpec{
		EndpointsSelection: api.EndpointsSelection{
			EndpointSelector: "lbl == 'lbl-val'",
			Namespaces: &api.NamesAndLabelsMatch{
				Selector: "endpoint-namespace-selector",
			},
			ServiceAccounts: &api.NamesAndLabelsMatch{
				Selector: "serviceaccount-selector",
			},
		},
	},
	EndpointsNumTotal:                     1,
	EndpointsNumIngressProtected:          10,
	EndpointsNumEgressProtected:           100,
	EndpointsNumIngressFromInternet:       1000,
	EndpointsNumEgressToInternet:          9000,
	EndpointsNumIngressFromOtherNamespace: 900,
	EndpointsNumEgressToOtherNamespace:    90,
	EndpointsNumEnvoyEnabled:              9,
	Endpoints: []api.EndpointsReportEndpoint{
		EndpointSample,
	},
}

/*
Returns rendered text for given text-template and data struct input.
*/
func RenderTemplate(reportTemplateText string, reportData api.ReportData) (rendered string, ret error) {
	defer func() {
		if perr := recover(); perr != nil {
			ret = fmt.Errorf("%v", perr)
		}
	}()

	fnmp := template.FuncMap{
		"join": joinResourceIds,
	}
	templ, err := template.New("report-template").Funcs(fnmp).Parse(reportTemplateText)
	if err != nil {
		return rendered, err
	}

	var b bytes.Buffer
	err = templ.Execute(&b, reportData)
	if err != nil {
		return rendered, err
	}
	rendered = b.String()

	return rendered, nil
}

/*
Join a list of ResourceID similar to  strings.Join() with a separator, capping maximum number of list
entries to avoid running into a huge list.
*/
func joinResourceIds(resources interface{}, sep string, max ...int) (joined string, ret error) {
	// First verify that right resource type is passed.
	if reflect.TypeOf(resources).Kind() != reflect.Slice {
		return joined, fmt.Errorf("Resource used with join is not a Slice")
	}

	res := reflect.ValueOf(resources)
	maxResources := res.Len()
	// Check if maximum resource count is specified.
	if len(max) > 0 {
		// Use the first value.
		maxResources = max[0]
	}

	buf := new(bytes.Buffer)
	for i := 0; i < maxResources; i++ {
		if i != 0 {
			buf.WriteString(sep)
		}

		fmt.Fprintf(buf, "%s", res.Index(i).Interface())
	}

	return buf.String(), nil
}
