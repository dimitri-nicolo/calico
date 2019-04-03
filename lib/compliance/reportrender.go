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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

////////////////////////////////////////////////////////////
var st = metav1.Unix(1554076800, 0)
var et = metav1.Time{st.Add(time.Hour * 10)}
var sel = "lbl == 'lbl-val'"
var selName = "sample-sel"
var epNum = 10
var resName = "sample-res"
var nsName = "sample-ns"
var kindName = "sample-kind"
var ResourceId = api.ResourceID{
	TypeMeta: metav1.TypeMeta{
		Kind: kindName,
	},
	Name:      resName,
	Namespace: nsName,
}

// Used by validator for template function validation.
var EndpointSample = api.EndpointsReportEndpoint{
	ID:               ResourceId,
	IngressProtected: false,
	EgressProtected:  true,
	EnvoyEnabled:     false,
	AppliedPolicies:  []api.ResourceID{ResourceId, ResourceId},
	Services:         []api.ResourceID{ResourceId, ResourceId},
}

// ReportDataSample is used by ReportTemplate validator.
var ReportDataSample = api.ReportData{
	StartTime: st,
	EndTime:   et,
	ReportSpec: api.ReportSpec{
		EndpointsSelection: api.EndpointsSelection{
			EndpointSelector: sel,
			Namespaces: &api.NamesAndLabelsMatch{
				Selector: selName,
			},
			ServiceAccounts: &api.NamesAndLabelsMatch{
				Selector: selName,
			},
		},
	},
	EndpointsNumTotal:                     epNum,
	EndpointsNumIngressProtected:          epNum,
	EndpointsNumEgressProtected:           epNum,
	EndpointsNumIngressFromInternet:       epNum,
	EndpointsNumEgressToInternet:          epNum,
	EndpointsNumIngressFromOtherNamespace: epNum,
	EndpointsNumEgressToOtherNamespace:    epNum,
	EndpointsNumEnvoyEnabled:              epNum,
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
			ret = fmt.Errorf("Template rendering error: %v", perr)
		}
	}()

	fnmp := template.FuncMap{
		"joinResources": joinResourceIds,
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
		return joined, fmt.Errorf("Resource used with joinResources is not a Slice")
	}

	res := reflect.ValueOf(resources)
	if res.Len() > 0 {
		if res.Index(0).Kind() != reflect.Struct {
			return joined, fmt.Errorf("Resource used with joinResources is not a Slice of Struct")
		}
	}

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

		kind := res.Index(i).FieldByName("Kind")
		name := res.Index(i).FieldByName("Name")
		if !kind.IsValid() || !name.IsValid() {
			return joined, fmt.Errorf("Resource used with joinResources doesn't contain Kind/Name")
		}
		namespace := res.Index(i).FieldByName("Namespace")

		// printing: kind(namespace/name)
		fmt.Fprintf(buf, "%s(", kind)
		if namespace.Len() > 0 {
			fmt.Fprintf(buf, "%s/", namespace)
		}
		fmt.Fprintf(buf, "%s)", name)
	}
	joined = buf.String()

	return joined, nil
}
