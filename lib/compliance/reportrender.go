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
	"encoding/json"
	"fmt"
	"reflect"
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/apis/audit"

	yaml "github.com/projectcalico/go-yaml-wrapper"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

var (
	// Exposed to be used by UT code.
	ResourceIdSample = api.ResourceID{
		TypeMeta: metav1.TypeMeta{
			Kind:       "sample-kind",
			APIVersion: "projectcalico.org/v3",
		},
		Name:      "sample-res",
		Namespace: "sample-ns",
	}
	EndpointSample = api.EndpointsReportEndpoint{
		Endpoint:         ResourceIdSample,
		IngressProtected: false,
		EgressProtected:  true,
		EnvoyEnabled:     false,
		AppliedPolicies:  []api.ResourceID{ResourceIdSample, ResourceIdSample},
		Services:         []api.ResourceID{ResourceIdSample, ResourceIdSample},
	}
	AuditEventSample = audit.Event{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Event",
			APIVersion: "audit.k8s.io/v1beta1",
		},
		Level:      "Metadata",
		AuditID:    "1-2-3-4-5",
		Stage:      "RequestReceived",
		RequestURI: "/api/v1/foo/bar",
		Verb:       "list",
		User: audit.UserInfo{
			Username: "userFoo",
			Groups:   []string{"groupFoo"},
		},
		ImpersonatedUser: &audit.UserInfo{
			Username: "imporUserFoo",
			Groups:   []string{"imperGroupFoo"},
		},
		SourceIPs: []string{"192.168.1.2"},
		ObjectRef: &audit.ObjectReference{
			Name:       "oRef",
			Namespace:  "default",
			Resource:   "fooBarResource",
			APIVersion: "v1",
		},
		ResponseStatus: &metav1.Status{
			Status: "k8s-audit-report-resp-status",
		},
		RequestObject: &runtime.Unknown{
			/*
				TypeMeta: runtime.TypeMeta{
					Kind:       "Request",
					APIVersion: "request/v1",
				},
			*/
			Raw:         []byte(`{"reqFoo": "reqBar"}`),
			ContentType: "application/json",
		},
		ResponseObject: &runtime.Unknown{
			/*
				TypeMeta: runtime.TypeMeta{
					Kind:       "Response",
					APIVersion: "response/v1",
				},
			*/
			Raw:         []byte(`{"respFoo": "respBar"}`),
			ContentType: "application/json",
		},
		RequestReceivedTimestamp: metav1.UnixMicro(1554076800, 0),
		StageTimestamp:           metav1.UnixMicro(1554112800, 0),
		Annotations:              map[string]string{"foo": "bar"},
	}

	// ReportDataSample is used by ReportTemplate validator.
	ReportDataSample = api.ReportData{
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
		EndpointsSummary: api.EndpointsSummary{
			NumTotal:                     1,
			NumIngressProtected:          10,
			NumEgressProtected:           100,
			NumIngressFromInternet:       1000,
			NumEgressToInternet:          9000,
			NumIngressFromOtherNamespace: 900,
			NumEgressToOtherNamespace:    90,
			NumEnvoyEnabled:              9,
		},
		Endpoints: []api.EndpointsReportEndpoint{
			EndpointSample,
		},
		AuditEvents: []audit.Event{
			AuditEventSample,
		},
	}
)

//
// Returns rendered text for given text-template and data struct input.
//
func RenderTemplate(reportTemplateText string, reportData *api.ReportData) (rendered string, ret error) {
	defer func() {
		if perr := recover(); perr != nil {
			ret = fmt.Errorf("%v", perr)
		}
	}()

	fnmp := template.FuncMap{
		"join": joinResourceIds,
		"json": jsonify,
		"yaml": yamlify,
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

//
// Join a list of ResourceID similar to  strings.Join() with a separator, capping maximum number of list
// entries to avoid running into a huge list.
//
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

//
// Print indented-JSON for a given struct.
//
func jsonify(resource interface{}) (string, error) {
	const prefix = ""
	const indent = "  "

	jsoned, err := json.MarshalIndent(resource, prefix, indent)
	if err != nil {
		return "", err
	}

	return string(jsoned), nil
}

//
// Print YAML for a given struct.
//
func yamlify(resource interface{}) (string, error) {
	yamled, err := yaml.Marshal(resource)
	if err != nil {
		return "", err
	}

	return string(yamled), nil
}
