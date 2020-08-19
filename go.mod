module github.com/projectcalico/cni-plugin

go 1.14

require (
	github.com/Microsoft/hcsshim v0.8.6
	github.com/buger/jsonparser v0.0.0-20180808090653-f4dd9f5a6b44
	github.com/containernetworking/cni v0.7.1
	github.com/containernetworking/plugins v0.8.5
	github.com/emicklei/go-restful v2.11.1+incompatible // indirect
	github.com/go-openapi/spec v0.19.4 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.3.1 // indirect
	github.com/howeyc/fsnotify v0.9.0
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/juju/clock v0.0.0-20180808021310-bab88fc67299
	github.com/juju/errors v0.0.0-20180806074554-22422dad46e1
	github.com/juju/mutex v0.0.0-20180619145857-d21b13acf4bf
	github.com/juju/retry v0.0.0-20180821225755-9058e192b216 // indirect
	github.com/juju/testing v0.0.0-20190723135506-ce30eb24acd2 // indirect
	github.com/juju/utils v0.0.0-20180820210520-bf9cc5bdd62d // indirect
	github.com/juju/version v0.0.0-20180108022336-b64dbd566305 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/mcuadros/go-version v0.0.0-20190308113854-92cdf37c5b75
	github.com/natefinch/atomic v0.0.0-20150920032501-a62ce929ffcc
	github.com/nmrshll/go-cp v0.0.0-20180115193924-61436d3b7cfa
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.1
	github.com/projectcalico/libcalico-go v1.7.2
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/common v0.6.0
	github.com/rakelkar/gonetsh v0.0.0-20190930180311-e5c5ffe4bdf0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/vishvananda/netlink v0.0.0-20181108222139-023a6dafdcdf
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
	golang.org/x/net v0.0.0-20200202094626-16171245cfb2
	golang.org/x/sys v0.0.0-20200202164722-d101bd2416d5
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/grpc v1.26.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
	k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6
)

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20200818223838-6dd792140b9c
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
)
