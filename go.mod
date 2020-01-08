module github.com/projectcalico/node

go 1.12

require (
	github.com/Microsoft/SDN v0.0.0-20181031164916-0d7593e5c8d4 // indirect
	github.com/kelseyhightower/confd v0.0.0-00010101000000-000000000000
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.1
	github.com/projectcalico/cni-plugin v3.8.2+incompatible
	github.com/projectcalico/felix v0.0.0-00010101000000-000000000000
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/projectcalico/pod2daemon v3.8.2+incompatible // indirect
	github.com/sirupsen/logrus v1.4.2
	gopkg.in/square/go-jose.v2 v2.2.3-0.20190111193340-cbf0fd6a984a // indirect
	k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/client-go v11.0.0+incompatible
)

replace (
	github.com/Microsoft/SDN => github.com/Microsoft/SDN v0.0.0-20181031164916-0d7593e5c8d4
	github.com/bronze1man/goStrongswanVici => github.com/tigera/goStrongswanVici v0.0.0-20180704141420-9b6fdd821dbe
	github.com/containernetworking/cni => github.com/containernetworking/cni v0.0.0-20180705210735-e67bb289cccf
	github.com/kelseyhightower/confd => github.com/tigera/confd-private v0.0.0-20200103201545-ee6e5a003174
	github.com/projectcalico/cni-plugin => github.com/tigera/cni-plugin-private v0.0.0-20191130192731-511e1d4a1416
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20191220191724-c757233f7c16
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200103165626-2c83fde7c5ce
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20191023103646-6c51073f7cfc
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	github.com/vishvananda/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320

	// Pin client-go to the v12.0.0 to avoid interdependency among other k8s.io/* packages bringing down the version.
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
)
