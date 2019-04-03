// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package report

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
		os.Setenv(reportNameEnv, reportName)
		os.Setenv(reportStartEnv, start)
		os.Setenv(reportEndEnv, end)

		By("validating the environments parsed correct")
		cfg, err := readReportConfigFromEnv()
		Expect(cfg).ToNot(BeNil())
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.Name).To(Equal(reportName))
		Expect(cfg.Report).To(BeNil())
		Expect(cfg.ReportType).To(BeNil())
		Expect(cfg.Start.Unix()).To(Equal(now.Unix()))
		Expect(cfg.End.Unix()).To(Equal(nowPlusHour.Unix()))
	})

	It("should error with missing configuration", func() {
		By("parsing with no config")
		os.Unsetenv(reportNameEnv)
		os.Unsetenv(reportStartEnv)
		os.Unsetenv(reportEndEnv)
		cfg, err := readReportConfigFromEnv()
		Expect(cfg).To(BeNil())
		Expect(err).To(HaveOccurred())

		By("parsing with no config")
		os.Setenv(reportNameEnv, "")
		os.Setenv(reportStartEnv, "")
		os.Setenv(reportEndEnv, "")
		cfg, err = readReportConfigFromEnv()
		Expect(cfg).To(BeNil())
		Expect(err).To(HaveOccurred())

		By("parsing with missing report name")
		os.Unsetenv(reportNameEnv)
		os.Setenv(reportStartEnv, start)
		os.Setenv(reportEndEnv, end)
		cfg, err = readReportConfigFromEnv()
		Expect(cfg).To(BeNil())
		Expect(err).To(HaveOccurred())

		By("parsing with missing start time")
		os.Setenv(reportNameEnv, reportName)
		os.Unsetenv(reportStartEnv)
		os.Setenv(reportEndEnv, end)
		cfg, err = readReportConfigFromEnv()
		Expect(cfg).To(BeNil())
		Expect(err).To(HaveOccurred())

		By("parsing with missing end time")
		os.Setenv(reportNameEnv, reportName)
		os.Setenv(reportStartEnv, start)
		os.Unsetenv(reportEndEnv)
		cfg, err = readReportConfigFromEnv()
		Expect(cfg).To(BeNil())
		Expect(err).To(HaveOccurred())
	})

	It("should error with invalid configuration", func() {
		By("parsing with invalid start time")
		os.Setenv(reportNameEnv, reportName)
		os.Setenv(reportStartEnv, "this is not a valid time")
		os.Setenv(reportEndEnv, end)
		cfg, err := readReportConfigFromEnv()
		Expect(cfg).To(BeNil())
		Expect(err).To(HaveOccurred())

		By("parsing with invalid end time")
		os.Setenv(reportNameEnv, reportName)
		os.Setenv(reportStartEnv, start)
		os.Setenv(reportEndEnv, "this is not a valid time")
		cfg, err = readReportConfigFromEnv()
		Expect(cfg).To(BeNil())
		Expect(err).To(HaveOccurred())

		By("parsing with end time before start time")
		os.Setenv(reportNameEnv, reportName)
		os.Setenv(reportStartEnv, end)
		os.Setenv(reportEndEnv, start)
		cfg, err = readReportConfigFromEnv()
		Expect(cfg).To(BeNil())
		Expect(err).To(HaveOccurred())
	})
})
