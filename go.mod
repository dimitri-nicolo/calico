module github.com/projectcalico/app-policy

go 1.15

require (
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/envoyproxy/go-control-plane v0.9.8
	github.com/envoyproxy/protoc-gen-validate v0.4.1 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/gogo/googleapis v1.4.1
	github.com/gogo/protobuf v1.3.2
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/libcalico-go v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.7.0
	golang.org/x/net v0.0.0-20210520170846-37e1c6afe023
	google.golang.org/genproto v0.0.0-20201110150050-8816d57aaa9a
	google.golang.org/grpc v1.27.1
	k8s.io/client-go v0.22.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210817084001-7fbd8d59e5b8 // indirect
)

replace (
	// Replace the envoy data-plane-api dependency with the projectcalico fork that includes the generated
	// go bindings for the API. Upstream only includes the protobuf definitions, so we need to fork in order to
	// supply the go code.
	github.com/envoyproxy/data-plane-api => github.com/projectcalico/data-plane-api v0.0.0-20210121211707-a620ff3c8f7e

	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20211126151825-2cc2008d4c6c
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
)
