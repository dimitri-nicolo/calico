module github.com/projectcalico/node

go 1.12

require (
	github.com/Microsoft/SDN v0.0.0-20181031164916-0d7593e5c8d4 // indirect
	github.com/coreos/etcd v3.3.13+incompatible // indirect
	github.com/google/gopacket v1.1.17 // indirect
	github.com/kelseyhightower/confd v0.16.0
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/projectcalico/cni-plugin v3.8.2+incompatible
	github.com/projectcalico/felix v0.0.0-00010101000000-000000000000
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/projectcalico/pod2daemon v3.8.2+incompatible // indirect
	github.com/sirupsen/logrus v1.4.2
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/client-go v8.0.0+incompatible
)

replace (
	github.com/Microsoft/SDN => github.com/Microsoft/SDN v0.0.0-20181031164916-0d7593e5c8d4
	github.com/bronze1man/goStrongswanVici => github.com/tigera/goStrongswanVici v0.0.0-20180704141420-9b6fdd821dbe
	github.com/containernetworking/cni => github.com/containernetworking/cni v0.0.0-20180705210735-e67bb289cccf
	github.com/kelseyhightower/confd => github.com/tigera/confd-private v0.0.0-20190921131451-f2cfad4654a0
	github.com/projectcalico/cni-plugin => github.com/tigera/cni-plugin-private v0.0.0-20190918161642-491d55d34d7e
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20190919144311-4759477d655a
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20190920203615-2ac87253de27
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20190917002945-931ed2638eb9
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v0.0.0-20180627202928-fc9bbf2f57995271c5cd6911ede7a2ebc5ea7c6f
	github.com/vishvananda/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320
)
