module github.com/projectcalico/cni-plugin

go 1.16

require (
	github.com/Microsoft/hcsshim v0.8.6
	github.com/buger/jsonparser v1.0.0
	github.com/containernetworking/cni v0.8.0
	github.com/containernetworking/plugins v0.8.5
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/gofrs/flock v0.8.0
	github.com/gogo/protobuf v1.3.2
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/howeyc/fsnotify v0.9.0
	github.com/juju/clock v0.0.0-20190205081909-9c5c9712527c
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/mutex v0.0.0-20180619145857-d21b13acf4bf
	github.com/juju/testing v0.0.0-20200608005635-e4eedbc6f7aa // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/mcuadros/go-version v0.0.0-20190308113854-92cdf37c5b75
	github.com/natefinch/atomic v0.0.0-20150920032501-a62ce929ffcc
	github.com/nmrshll/go-cp v0.0.0-20180115193924-61436d3b7cfa
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/libcalico-go v1.7.2
	github.com/prometheus/common v0.26.0
	github.com/rakelkar/gonetsh v0.0.0-20190930180311-e5c5ffe4bdf0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/tigera/api v0.0.0-20211025235918-67c9d2e86873
	github.com/vishvananda/netlink v1.1.1-0.20210703095558-21f2c55a7727
	go.etcd.io/etcd v0.5.0-alpha.5.0.20201125193152-8a03d2e9614b
	golang.org/x/sys v0.0.0-20210616094352-59db8d763f22
	golang.org/x/tools v0.1.2 // indirect
	google.golang.org/grpc v1.27.1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/client-go v0.22.0
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176
)

replace (
	github.com/Microsoft/hcsshim => github.com/projectcalico/hcsshim v0.8.9-calico
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20211026005712-dcbb50694290
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	google.golang.org/grpc => google.golang.org/grpc v1.27.1

	k8s.io/api => k8s.io/api v0.21.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.0
	k8s.io/client-go => k8s.io/client-go v0.21.0
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
)
