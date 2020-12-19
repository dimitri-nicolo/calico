module github.com/tigera/license-agent

go 1.13

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20201215061722-3d90f1306e73

require (
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/client_golang v1.0.0
	github.com/sirupsen/logrus v1.4.2
	github.com/tigera/licensing v1.0.1-0.20201218173202-d766f2323fee
)
