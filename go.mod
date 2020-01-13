module github.com/projectcalico/node

go 1.12

require (
	github.com/kelseyhightower/confd v0.0.0-00010101000000-000000000000
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.1
	github.com/projectcalico/cni-plugin v3.8.2+incompatible
	github.com/projectcalico/felix v3.8.5+incompatible
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/projectcalico/pod2daemon v3.8.2+incompatible // indirect
	github.com/sirupsen/logrus v1.4.2
	gopkg.in/fsnotify/fsnotify.v1 v1.4.7

	// k8s.io/api v1.16.3 is at 16d7abae0d2a
	k8s.io/api v0.0.0-20191114100352-16d7abae0d2a

	// k8s.io/apimachinery 1.16.3 is at 72ed19daf4bb
	k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb

	// k8s.io/client-go 1.16.3 is at 6c5935290e33
	k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
)

replace (
	github.com/Microsoft/SDN => github.com/Microsoft/SDN v0.0.0-20181031164916-0d7593e5c8d4
	github.com/bronze1man/goStrongswanVici => github.com/tigera/goStrongswanVici v0.0.0-20180704141420-9b6fdd821dbe
	github.com/containernetworking/cni => github.com/containernetworking/cni v0.0.0-20180705210735-e67bb289cccf
	github.com/kelseyhightower/confd => github.com/tigera/confd-private v0.0.0-20200103201545-ee6e5a003174
	github.com/projectcalico/cni-plugin => github.com/tigera/cni-plugin-private v0.0.0-20200111003839-08326be6b392
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20200111223629-729cb7baf6f1
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200110190915-9fa812d46e44
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.0.0-20200103211238-4018e3107793
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	github.com/vishvananda/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320
)
