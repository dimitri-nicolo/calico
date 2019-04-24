// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package config

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"
	"time"
)

var (
	now         = time.Now()
	nowPlusHour = now.Add(time.Hour)
	reportName  = "my-report"
	start       = now.Format(time.RFC3339)
	end         = nowPlusHour.Format(time.RFC3339)
)

var _ = Describe("Load config from environments", func() {
	It("should parse valid configuration", func() {
		By("parsing with valid config")
		os.Setenv(ReportNameEnv, reportName)
		os.Setenv(ReportStartEnv, start)
		os.Setenv(ReportEndEnv, end)

		By("validating the environments parsed correct")
		cfg, err := LoadConfig()
		Expect(cfg).ToNot(BeNil())
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.ReportName).To(Equal(reportName))
		Expect(cfg.ParsedReportStart.Unix()).To(Equal(now.Unix()))
		Expect(cfg.ParsedReportEnd.Unix()).To(Equal(nowPlusHour.Unix()))
	})

	It("should error with invalid configuration", func() {
		By("parsing with invalid start time")
		os.Setenv(ReportNameEnv, reportName)
		os.Setenv(ReportStartEnv, "this is not a valid time")
		os.Setenv(ReportEndEnv, end)
		cfg, err := LoadConfig()
		Expect(err).To(HaveOccurred())
		Expect(cfg).To(BeNil())

		By("parsing with invalid end time")
		os.Setenv(ReportNameEnv, reportName)
		os.Setenv(ReportStartEnv, start)
		os.Setenv(ReportEndEnv, "this is not a valid time")
		cfg, err = LoadConfig()
		Expect(cfg).To(BeNil())
		Expect(err).To(HaveOccurred())

		By("parsing with end time before start time")
		os.Setenv(ReportNameEnv, reportName)
		os.Setenv(ReportStartEnv, end)
		os.Setenv(ReportEndEnv, start)
		cfg, err = LoadConfig()
		Expect(cfg).To(BeNil())
		Expect(err).To(HaveOccurred())
	})
})
