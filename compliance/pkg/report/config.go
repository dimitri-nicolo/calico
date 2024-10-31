// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.
package report

import (
	"context"

	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/projectcalico/calico/compliance/pkg/config"
	"github.com/projectcalico/calico/compliance/pkg/datastore"
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
	client := datastore.MustGetClientSet()

	reportCfg.Report, err = client.GlobalReports().Get(context.Background(), reportCfg.ReportName, metav1.GetOptions{})
	if err != nil {
		log.WithError(err).Panicf("Global report %s not found.", reportCfg.ReportName)
	}

	reportCfg.ReportType, err = client.GlobalReportTypes().Get(context.Background(), reportCfg.Report.Spec.ReportType, metav1.GetOptions{})
	if err != nil {
		log.Panicf("Global report-type %s not found.", reportCfg.Report.Spec.ReportType)
	}

	return reportCfg
}

// Use the standard config loader, but also check that a report name has been specified.
func mustReadReportConfigFromEnv() *Config {
	base := config.MustLoadConfig()

	// The ReportName is mandatory for the Reporter.
	if base.ReportName == "" {
		log.Panic("Report-name environment variable TIGERA_COMPLIANCE_REPORT_NAME cannot be empty.")
	}

	return &Config{
		Config: *base,
	}
}
