module github.com/tigera/license-agent

go 1.16

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20211026184131-9e4915b8bac2

require (
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/client_golang v1.7.1
	github.com/sirupsen/logrus v1.7.0
	github.com/tigera/api v0.0.0-20211026170921-dc6d9aa2cf7c
	github.com/tigera/licensing v1.0.1-0.20211026184521-75dd4b3a7137
)

replace k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
