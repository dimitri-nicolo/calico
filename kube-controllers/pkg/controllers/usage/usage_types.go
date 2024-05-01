package usage

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

// LicenseUsageReportData represents the data of license usage reports. This data appears as a serialized and base64-encoded
// message within the LicenseUsageReport. It is stored in a serialized form on the LicenseUsageReport so that there is no
// ambiguity for clients on how to validate the HMAC of the LicenseUsageReport: they simply treat the serialized report
// data as the 'message'.
// IMPORTANT: Do not make any breaking changes (e.g. edits or deletions) to the existing fields in this struct. Doing so
// could impact backwards-compatibility, and cause deserialization of older reports to have missing fields.
type LicenseUsageReportData struct {
	// The start of the observation window.
	IntervalStart time.Time `json:"intervalStart"`

	// The end of the observation window.
	IntervalEnd time.Time `json:"intervalEnd"`

	// The vCPU-related usage stats collected during the interval.
	VCPUs Stats `json:"vCPUs"`

	// The node-related usage stats collected during the interval.
	Nodes Stats `json:"nodes"`

	// Identifies the subject of the usage monitoring.
	SubjectUID string `json:"subjectUID,omitempty"`

	// Identifies the license applied during the usage monitoring.
	LicenseUID string `json:"licenseUID,omitempty"`

	// Identifies the last report that was published by the usage reporter.
	LastPublishedReportUID string `json:"lastPublishedReportUID,omitempty"`

	// Seconds that the reporter has been up for.
	ReporterUptime int `json:"reporterUptime"`
}

type Stats struct {
	// The minimum value observed during the interval.
	Min int `json:"min"`
	// The maximum value observed during the interval.
	Max int `json:"max"`
}

// ToMessage converts the LicenseUsageReportData to a string, suitable for use as a 'message' in HMAC terms.
func (d *LicenseUsageReportData) ToMessage() (string, error) {
	reportDataBytes, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(reportDataBytes), nil
}

func NewLicenseUsageReportDataFromMessage(message string) (LicenseUsageReportData, error) {
	reportDataBytes, err := base64.StdEncoding.DecodeString(message)
	if err != nil {
		return LicenseUsageReportData{}, err
	}

	var reportData LicenseUsageReportData
	err = json.Unmarshal(reportDataBytes, &reportData)
	if err != nil {
		return LicenseUsageReportData{}, err
	}

	return reportData, nil
}
