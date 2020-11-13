module github.com/tigera/envoy-collector

go 1.13

require (
	github.com/gogo/googleapis v1.3.2
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/hpcloud/tail v1.0.0
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/lyft/protoc-gen-star v0.4.14
	github.com/lyft/protoc-gen-validate v0.1.0
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/net v0.0.0-20200202094626-16171245cfb2
	google.golang.org/grpc v1.26.0
)

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20200924131826-5c5f99461e0c
