module github.com/projectcalico/app-policy

go 1.12

require (
	github.com/docopt/docopt-go v0.0.0-20160216232012-784ddc588536
	github.com/envoyproxy/data-plane-api v0.0.0-20190513203724-4a93c6d2d917 // indirect
	github.com/gogo/googleapis v1.0.0
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/lyft/protoc-gen-validate v0.0.6 // indirect
	github.com/onsi/gomega v1.7.0
	github.com/philhofer/fwd v1.0.0 // indirect
	github.com/pquerna/ffjson v0.0.0-20190813045741-dac163c6c0a9 // indirect
	github.com/projectcalico/go-yaml v0.0.0-20161201183616-955bc3e451ef // indirect
	github.com/projectcalico/libcalico-go v0.0.0-00000000000000-000000000000
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/tinylib/msgp v1.1.0 // indirect
	github.com/ugorji/go v0.0.0-20171019201919-bdcc60b419d1 // indirect
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9
	google.golang.org/grpc v1.23.1
	k8s.io/client-go v12.0.0+incompatible // indirect
)

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200130203734-85760318d620
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
)
