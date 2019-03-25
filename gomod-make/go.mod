module github.com/tigera/compliance

require (
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.0
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/apimachinery v0.0.0-20190322184251-46b17dfe5118
	k8s.io/klog v0.2.0 // indirect
	github.com/projectcalico/libcalico-go v0.0.0
)

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v2.1.1-0.20190307140245-b57de3f3848a+incompatible
