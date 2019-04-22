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
	"strings"
	"text/template"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/apis/audit"

	"github.com/Masterminds/sprig"
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

	ReportDataNilNamespace = api.ReportData{
		ReportName: "nil-namespace",
		ReportSpec: api.ReportSpec{
			EndpointsSelection: api.EndpointsSelection{
				EndpointSelector: "lbl == 'lbl-val'",
				Namespaces:       nil,
			},
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

	templ, err := template.New("report-template").Funcs(templateFuncs()).Parse(reportTemplateText)
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

// yamlify prints YAML for a given struct.
func yamlify(resource interface{}) (string, error) {
	yamled, err := yaml.Marshal(resource)
	if err != nil {
		return "", err
	}

	return string(yamled), nil
}

// formatDate returns the date in the specified format
func getFormatDateFn(format string) func(date interface{}) string {
	return func(date interface{}) string {
		switch d := date.(type) {
		case time.Time:
			return d.Format(format)
		case *time.Time:
			if d == nil {
				return "nil"
			}
			return d.Format(format)
		case metav1.Time:
			return d.Format(format)
		case *metav1.Time:
			if d == nil {
				return "nil"
			}
			return d.Format(format)
		}
		return fmt.Sprint(date)
	}
}

func templateFuncs() template.FuncMap {
	// Use the functions defined by sprig and add a couple more.
	funcs := sprig.GenericFuncMap()

	// Add YAML conversion, naming as per toJson.
	funcs["toYaml"] = yamlify

	// Add a joinN function which joins an array of strings up to a max number of elements. We utilize the
	// sprig toStrings() method to convert the final arg to a string slice.
	toStrings := funcs["toStrings"].(func(interface{}) []string)
	funcs["joinN"] = func(sep string, max int, v interface{}) string {
		s := toStrings(v)
		if len(s) > max {
			s = s[:max]
		}
		return strings.Join(s, sep)
	}

	// Add a useful time formats, we can add more later if required.
	funcs["dateRfc3339"] = getFormatDateFn(time.RFC3339)

	return template.FuncMap(funcs)
}
