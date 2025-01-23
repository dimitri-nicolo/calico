package calico

import "github.com/projectcalico/calico/release/internal/hashreleaseserver"

type EnterpriseOption func(*EnterpriseManager) error

func WithChartVersion(version string) EnterpriseOption {
	return func(r *EnterpriseManager) error {
		r.chartVersion = version
		return nil
	}
}

func WithEnterpriseHashrelease(hashrelease hashreleaseserver.EnterpriseHashrelease, cfg hashreleaseserver.Config) EnterpriseOption {
	return func(r *EnterpriseManager) error {
		r.enterpriseHashrelease = hashrelease
		r.hashrelease = hashrelease.Hashrelease
		r.hashreleaseConfig = cfg
		return nil
	}
}

func WithPublishWindowsArchive(publish bool) EnterpriseOption {
	return func(r *EnterpriseManager) error {
		r.publishWindowsArchive = publish
		return nil
	}
}

func WithPublishCharts(publish bool) EnterpriseOption {
	return func(r *EnterpriseManager) error {
		r.publishCharts = publish
		return nil
	}
}

func WithHelmRegistry(registry string) EnterpriseOption {
	return func(r *EnterpriseManager) error {
		r.helmRegistry = registry
		return nil
	}
}
