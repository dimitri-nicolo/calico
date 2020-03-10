module github.com/projectcalico/node

go 1.12

require (
	github.com/Microsoft/SDN v0.0.0-20181031164916-0d7593e5c8d4
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
	k8s.io/api v0.17.2

	// k8s.io/apimachinery 1.16.3 is at 72ed19daf4bb
	k8s.io/apimachinery v0.17.2

	// k8s.io/client-go 1.16.3 is at 6c5935290e33
	k8s.io/client-go v0.17.2
)

replace (
	github.com/Microsoft/SDN => github.com/Microsoft/SDN v0.0.0-20181031164916-0d7593e5c8d4
	github.com/bronze1man/goStrongswanVici => github.com/tigera/goStrongswanVici v0.0.0-20180704141420-9b6fdd821dbe
	github.com/containernetworking/cni => github.com/containernetworking/cni v0.0.0-20180705210735-e67bb289cccf
	github.com/kelseyhightower/confd => github.com/tigera/confd-private v0.0.0-20200310041910-3ee937049275
	github.com/projectcalico/cni-plugin => github.com/tigera/cni-plugin-private v0.0.0-20200307042119-0a890c792973
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20200304211117-5caba4aba4a3
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200306170834-701ab417e6cb
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.0.0-20200307163514-61b3e6fe48b8
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	github.com/vishvananda/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320
)
