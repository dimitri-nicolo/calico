module github.com/tigera/lma

go 1.12

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200103165626-2c83fde7c5ce // indirect

require (
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/olivere/elastic/v7 v7.0.6
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/projectcalico/libcalico-go v0.0.0-00010101000000-000000000000
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/apiserver v0.17.0
	k8s.io/client-go v0.17.0
)
