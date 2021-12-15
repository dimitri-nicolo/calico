package calico

import (
	"context"
	"time"

	"github.com/projectcalico/calico/licensing/client"
)

// LicenseMonitor is an interface which enables monitoring of license and feature enablement status.
type MockLicenseMonitorAllowAll struct{}

func (m MockLicenseMonitorAllowAll) GetFeatureStatus(string) bool           { return true }
func (m MockLicenseMonitorAllowAll) GetLicenseStatus() client.LicenseStatus { return client.Valid }
func (m MockLicenseMonitorAllowAll) MonitorForever(context.Context) error   { return nil }
func (m MockLicenseMonitorAllowAll) RefreshLicense(context.Context) error   { return nil }
func (m MockLicenseMonitorAllowAll) SetPollInterval(duration time.Duration) {}
func (m MockLicenseMonitorAllowAll) SetFeaturesChangedCallback(func())      {}
func (m MockLicenseMonitorAllowAll) SetStatusChangedCallback(f func(newLicenseStatus client.LicenseStatus)) {
}

type MockLicenseMonitorAllowNone struct{}

func (m MockLicenseMonitorAllowNone) GetFeatureStatus(string) bool           { return false }
func (m MockLicenseMonitorAllowNone) GetLicenseStatus() client.LicenseStatus { return client.Valid }
func (m MockLicenseMonitorAllowNone) MonitorForever(context.Context) error   { return nil }
func (m MockLicenseMonitorAllowNone) RefreshLicense(context.Context) error   { return nil }
func (m MockLicenseMonitorAllowNone) SetPollInterval(duration time.Duration) {}
func (m MockLicenseMonitorAllowNone) SetFeaturesChangedCallback(func())      {}
func (m MockLicenseMonitorAllowNone) SetStatusChangedCallback(f func(newLicenseStatus client.LicenseStatus)) {
}
