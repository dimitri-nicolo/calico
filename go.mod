module github.com/tigera/license-agent

go 1.12

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200319165815-dcfd07befeb2

require (
	github.com/kelseyhightower/envconfig v0.0.0-20180517194557-dd1402a4d99d
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/sirupsen/logrus v1.4.2
	github.com/tigera/licensing v2.5.1+incompatible
)
