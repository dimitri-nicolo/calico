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
	"text/template"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

var st = metav1.Unix(1554076800, 0)
var et = metav1.Time{st.Add(time.Hour * 10)}
var sel = "lbl == 'lbl-val'"
var name = "grt-sel"
var ep_num = 10

// ReportDataSample is used by ReportTemplate validator.
var ReportDataSample = api.ReportData{
	StartTime: st,
	EndTime:   et,
	ReportSpec: api.ReportSpec{
		EndpointsSelection: api.EndpointsSelection{
			EndpointSelector: sel,
			Namespaces: &api.NamesAndLabelsMatch{
				Selector: name,
			},
			ServiceAccounts: &api.NamesAndLabelsMatch{
				Selector: name,
			},
		},
	},
	EndpointsNumTotal:                     ep_num,
	EndpointsNumIngressProtected:          ep_num,
	EndpointsNumEgressProtected:           ep_num,
	EndpointsNumIngressFromInternet:       ep_num,
	EndpointsNumEgressToInternet:          ep_num,
	EndpointsNumIngressFromOtherNamespace: ep_num,
	EndpointsNumEgressToOtherNamespace:    ep_num,
	EndpointsNumEnvoyEnabled:              ep_num,
}

/*
Returns rendered text for given text-template and data struct input.
*/
func RenderTemplate(reportTemplateText string, reportData api.ReportData) (string, error) {
	var rendered string

	templ, err := template.New("report-template").Parse(reportTemplateText)
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
