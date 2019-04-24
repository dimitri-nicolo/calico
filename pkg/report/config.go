// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package report

import (
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	"github.com/tigera/compliance/pkg/config"
	"github.com/tigera/compliance/pkg/datastore"
)

// Extend the base config to also load in the Report and ReportType
type Config struct {
	config.Config

	// --- Loaded from Calico ---
	Report     *v3.GlobalReport
	ReportType *v3.GlobalReportType
}

func MustLoadReportConfig(cfg *config.Config) *Config {
	var err error
	reportCfg := mustReadReportConfigFromEnv()

	// Get the calico client and pull the named report and corresponding report type.
	client := datastore.MustGetCalicoClient()

	reportCfg.Report, err = client.GlobalReports().Get(reportCfg.ReportName, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	reportCfg.ReportType, err = client.GlobalReportTypes().Get(reportCfg.Report.Spec.ReportType, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	return reportCfg
}

// Use the standard config loader, but also check that a report name has been specified.
func mustReadReportConfigFromEnv() *Config {
	base := config.MustLoadConfig()

	// The ReportName is mandatory for the Reporter.
	if base.ReportName == "" {
		panic(errors.New("report name has not been specified in environments"))
	}

	return &Config{
		Config: *base,
	}
}
