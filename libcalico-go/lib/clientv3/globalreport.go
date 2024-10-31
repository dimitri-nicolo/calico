// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package clientv3

import (
	"context"

	log "github.com/sirupsen/logrus"
	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	cerrors "github.com/projectcalico/calico/libcalico-go/lib/errors"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	validator "github.com/projectcalico/calico/libcalico-go/lib/validator/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/watch"
)

// GlobalReportInterface has methods to work with GlobalReport resources.
type GlobalReportInterface interface {
	Create(ctx context.Context, res *apiv3.GlobalReport, opts options.SetOptions) (*apiv3.GlobalReport, error)
	Update(ctx context.Context, res *apiv3.GlobalReport, opts options.SetOptions) (*apiv3.GlobalReport, error)
	Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.GlobalReport, error)
	Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.GlobalReport, error)
	List(ctx context.Context, opts options.ListOptions) (*apiv3.GlobalReportList, error)
	Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error)
}

// globalReports implements GlobalReportInterface
type globalReports struct {
	client client
}

var (
	five    = 5
	fifty   = 50
	hundred = 100
)

// Create takes the representation of a GlobalReport and creates it.  Returns the stored
// representation of the GlobalReport, and an error, if there is any.
func (r globalReports) Create(ctx context.Context, res *apiv3.GlobalReport, opts options.SetOptions) (*apiv3.GlobalReport, error) {
	// Set reasonable defaults if certain fields are empty.
	if res.Spec.CIS != nil {
		// NumFailedTests
		if res.Spec.CIS.NumFailedTests == nil {
			res.Spec.CIS.NumFailedTests = &five
		}

		// HighThreshold
		if res.Spec.CIS.HighThreshold == nil {
			res.Spec.CIS.HighThreshold = &hundred
		}

		// MedThreshold
		if res.Spec.CIS.MedThreshold == nil {
			res.Spec.CIS.MedThreshold = &fifty
		}
	}

	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	// Validate GlobalReportType exists before create.
	if err := r.CheckGlobalReportTypeExists(ctx, res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Create(ctx, opts, apiv3.KindGlobalReport, res)
	if out != nil {
		return out.(*apiv3.GlobalReport), err
	}
	return nil, err
}

// Update takes the representation of a GlobalReport and updates it. Returns the stored
// representation of the GlobalReport, and an error, if there is any.
func (r globalReports) Update(ctx context.Context, res *apiv3.GlobalReport, opts options.SetOptions) (*apiv3.GlobalReport, error) {
	if err := validator.Validate(res); err != nil {
		return nil, err
	}

	// Validate GlobalReportType exists before update.
	if err := r.CheckGlobalReportTypeExists(ctx, res); err != nil {
		return nil, err
	}

	out, err := r.client.resources.Update(ctx, opts, apiv3.KindGlobalReport, res)
	if out != nil {
		return out.(*apiv3.GlobalReport), err
	}
	return nil, err
}

// Delete takes name of the GlobalReport and deletes it. Returns an error if one occurs.
func (r globalReports) Delete(ctx context.Context, name string, opts options.DeleteOptions) (*apiv3.GlobalReport, error) {
	out, err := r.client.resources.Delete(ctx, opts, apiv3.KindGlobalReport, noNamespace, name)
	if out != nil {
		return out.(*apiv3.GlobalReport), err
	}
	return nil, err
}

// Get takes name of the GlobalReport, and returns the corresponding GlobalReport object,
// and an error if there is any.
func (r globalReports) Get(ctx context.Context, name string, opts options.GetOptions) (*apiv3.GlobalReport, error) {
	out, err := r.client.resources.Get(ctx, opts, apiv3.KindGlobalReport, noNamespace, name)
	if out != nil {
		return out.(*apiv3.GlobalReport), err
	}
	return nil, err
}

// List returns the list of GlobalReport objects that match the supplied options.
func (r globalReports) List(ctx context.Context, opts options.ListOptions) (*apiv3.GlobalReportList, error) {
	res := &apiv3.GlobalReportList{}
	if err := r.client.resources.List(ctx, opts, apiv3.KindGlobalReport, apiv3.KindGlobalReportList, res); err != nil {
		return nil, err
	}
	return res, nil
}

// Watch returns a watch.Interface that watches the GlobalReports that match the
// supplied options.
func (r globalReports) Watch(ctx context.Context, opts options.ListOptions) (watch.Interface, error) {
	return r.client.resources.Watch(ctx, opts, apiv3.KindGlobalReport, nil)
}

// Check that GlobalReportType configuration referenced in the GlobalReport configuration exists.
func (c globalReports) CheckGlobalReportTypeExists(ctx context.Context, report *apiv3.GlobalReport) error {
	reportTypeName := report.Spec.ReportType

	_, err := c.client.GlobalReportTypes().Get(ctx, reportTypeName, options.GetOptions{})
	if err != nil {
		log.WithError(err).Debugf("ReportType(%s) configured with GlobalReport(%s) doesn't exist", reportTypeName, report.Name)

		if _, ok := err.(cerrors.ErrorResourceDoesNotExist); ok {
			return cerrors.ErrorValidation{
				ErroredFields: []cerrors.ErroredField{
					{
						Name:   "ReportType",
						Value:  reportTypeName,
						Reason: err.Error(),
					},
				},
			}
		}
	}

	return err
}
