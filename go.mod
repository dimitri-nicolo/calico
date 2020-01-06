module github.com/tigera/ingress-collector

go 1.12

require (
	github.com/envoyproxy/go-control-plane v0.9.1
	github.com/envoyproxy/protoc-gen-validate v0.1.0
	github.com/go-openapi/jsonreference v0.19.3 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/hpcloud/tail v1.0.0
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/lyft/protoc-gen-star v0.4.14
	github.com/onsi/ginkgo v1.6.0
	github.com/onsi/gomega v1.7.0
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/afero v1.2.2 // indirect
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553
	google.golang.org/grpc v1.23.0
	k8s.io/kube-openapi v0.0.0-20190918143330-0270cf2f1c1d // indirect
)

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20191231191334-8b565fb76d34
