module github.com/tigera/license-agent

go 1.16

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20211202183359-a69b8ac57bf8

require (
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/client_golang v1.7.1
	github.com/sirupsen/logrus v1.8.1
	github.com/tigera/api v0.0.0-20211202170222-d8128d06db71
	github.com/tigera/licensing v1.0.1-0.20211202191130-10676ef078af
)

replace k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20211110012726-3cc51fd1e909
