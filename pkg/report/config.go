// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package report

import (
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"

	"github.com/tigera/compliance/pkg/datastore"
)

const (
	reportNameEnv  = "TIGERA_COMPLIANCE_REPORT_NAME"
	reportStartEnv = "TIGERA_COMPLIANCE_START_TIME"
	reportEndEnv   = "TIGERA_COMPLIANCE_END_TIME"
)

var (
	// Parameterised for testing.
	getenv = os.Getenv
)

type Config struct {
	// --- Loaded from environment ---
	Name  string
	Start time.Time
	End   time.Time

	// --- Loaded from Calico ---
	Report     *v3.GlobalReport
	ReportType *v3.GlobalReportType
}

func MustLoadReportConfig() *Config {
	var err error
	rc, err := readReportConfigFromEnv()
	if err != nil {
		panic(err)
	}

	client := datastore.MustGetCalicoClient()

	rc.Report, err = client.GlobalReports().Get(rc.Name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	rc.ReportType, err = client.GlobalReportTypes().Get(rc.Report.Spec.ReportType, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	return rc
}

func readReportConfigFromEnv() (*Config, error) {
	// Determine the name of the report and the start and end time for the report.
	reportName := getenv(reportNameEnv)
	if reportName == "" {
		return nil, fmt.Errorf("no report name specified in environment %s", reportNameEnv)
	}

	startEnv := getenv(reportStartEnv)
	if startEnv == "" {
		return nil, fmt.Errorf("no report start time specified in environment %s", reportStartEnv)
	}
	start, err := time.Parse(time.RFC3339, startEnv)
	if err != nil {
		return nil, fmt.Errorf("report start time specified in environment %s is not RFC3339 formatted: %s", reportStartEnv, startEnv)
	}

	endEnv := getenv(reportEndEnv)
	if endEnv == "" {
		return nil, fmt.Errorf("no report end time specified in environment %s", reportEndEnv)
	}
	end, err := time.Parse(time.RFC3339, endEnv)
	if err != nil {
		return nil, fmt.Errorf("report end time specified in environment %s is not RFC3339 formatted: %s", reportEndEnv, endEnv)
	}

	if end.Before(start) {
		return nil, fmt.Errorf(
			"invalid report times, end time is before start time: %s (%s) < %s (%s)",
			endEnv, reportEndEnv, startEnv, reportStartEnv,
		)
	}

	return &Config{
		Name:  reportName,
		Start: start,
		End:   end,
	}, nil
}
