module github.com/tigera/ingress-collector

go 1.15

require (
	github.com/gogo/protobuf v1.3.2
	github.com/hpcloud/tail v1.0.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/sirupsen/logrus v1.7.0
	golang.org/x/net v0.0.0-20210224082022-3d97a244fca7
	google.golang.org/grpc v1.27.1
)

replace (
	// Replace the envoy data-plane-api dependency with the projectcalico fork that includes the generated
	// go bindings for the API. Upstream only includes the protobuf definitions, so we need to fork in order to
	// supply the go code.
	github.com/envoyproxy/data-plane-api => github.com/projectcalico/data-plane-api v0.0.0-20210121211707-a620ff3c8f7e
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210628142241-49d0211f9a0a
)
