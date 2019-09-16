module github.com/projectcalico/node

go 1.12

require (
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/aws/aws-sdk-go v1.23.22 // indirect
	github.com/bronze1man/goStrongswanVici v0.0.0-20190828090544-27d02f80ba40 // indirect
	github.com/containernetworking/cni v0.7.1 // indirect
	github.com/containernetworking/plugins v0.8.2 // indirect
	github.com/coreos/etcd v3.3.13+incompatible // indirect
	github.com/gavv/monotime v0.0.0-20190418164738-30dba4353424 // indirect
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/google/gopacket v1.1.17 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/kelseyhightower/confd v0.16.0
	github.com/kelseyhightower/memkv v0.1.1 // indirect
	github.com/natefinch/atomic v0.0.0-20150920032501-a62ce929ffcc // indirect
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/projectcalico/cni-plugin v3.8.2+incompatible
	github.com/projectcalico/felix v0.0.0-20190913210453-12f12e7ff6a3
	github.com/projectcalico/libcalico-go v0.0.0-20190909144507-d0b62f71c979
	github.com/projectcalico/pod2daemon v3.8.2+incompatible // indirect
	github.com/projectcalico/typha v0.0.0-20190909183257-4a0fda01b791 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/tigera/licensing v2.6.0-0.dev+incompatible // indirect
	github.com/tigera/nfnetlink v0.0.0-20190401090543-2623d65568be // indirect
	github.com/vishvananda/netlink v1.0.0 // indirect
	go.etcd.io/bbolt v1.3.3 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/client-go v8.0.0+incompatible
)

replace github.com/sirupsen/logrus => github.com/projectcalico/logrus v0.0.0-20180627202928-fc9bbf2f57995271c5cd6911ede7a2ebc5ea7c6f

replace github.com/projectcalico/felix => github.com/tigera/felix-private v2.6.0-0.dev.0.20190916204147-a2edee9cf851+incompatible

replace github.com/kelseyhightower/confd => github.com/tigera/confd-private v2.6.0-0.dev.0.20190802095729-650fd08ca116+incompatible

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20190913172157-338f687c53ee

replace github.com/vishvanada/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320
