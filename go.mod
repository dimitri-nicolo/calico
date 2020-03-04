module github.com/projectcalico/felix

go 1.12

require (
	github.com/Azure/go-autorest/autorest v0.9.3 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.8.1 // indirect
	github.com/Microsoft/go-winio v0.0.0-20190408173621-84b4ab48a507 // indirect
	github.com/Microsoft/hcsshim v0.0.0-20190408221605-063ae4a83d78
	github.com/aws/aws-sdk-go v1.13.54
	github.com/bronze1man/goStrongswanVici v0.0.0-20190828090544-27d02f80ba40
	github.com/containernetworking/cni v0.5.2
	github.com/davecgh/go-spew v1.1.1
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/gavv/monotime v0.0.0-20190418164738-30dba4353424
	github.com/go-ini/ini v0.0.0-20190327024845-3be5ad479f69
	github.com/gobuffalo/packr/v2 v2.0.9
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/google/gopacket v1.1.17
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gophercloud/gophercloud v0.4.0 // indirect
	github.com/gxed/GoEndian v0.0.0-20160916112711-0f5c6873267e // indirect
	github.com/gxed/eventfd v0.0.0-20160916113412-80a92cca79a8 // indirect
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/ipfs/go-log v0.0.0-20180611222144-5dc2060baaf8 // indirect
	github.com/ishidawataru/sctp v0.0.0-20191218070446-00ab2ac2db07
	github.com/jmespath/go-jmespath v0.0.0-20151117175822-3433f3ea46d9 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/libp2p/go-reuseport v0.0.1
	github.com/libp2p/go-sockaddr v0.0.0-20190411201116-52957a0228cc // indirect
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mattn/go-isatty v0.0.7 // indirect
	github.com/mipearson/rfw v0.0.0-20170619235010-6f0a6f3266ba
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.1
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/projectcalico/pod2daemon v0.0.0-20191223184832-a0e1c4693271
	github.com/projectcalico/typha v3.8.2+incompatible
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/smartystreets/goconvey v0.0.0-20190731233626-505e41936337 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/tigera/licensing v0.0.0-20200225180546-5125719fc8ad
	github.com/tigera/nfnetlink v0.0.0-20190401090543-2623d65568be
	github.com/vishvananda/netlink v0.0.0-20180501223456-f07d9d5231b9
	github.com/vishvananda/netns v0.0.0-20160430053723-8ba1072b58e0 // indirect
	github.com/whyrusleeping/go-logging v0.0.0-20170515211332-0457bb6b88fc // indirect
	golang.org/x/net v0.0.0-20191112182307-2180aed22343
	golang.org/x/sys v0.0.0-20191113165036-4c7a9d0fe056
	google.golang.org/grpc v1.26.0
	gopkg.in/ini.v1 v1.46.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/tchap/go-patricia.v2 v2.3.0

	// k8s.io/api v1.16.3 is at 16d7abae0d2a
	k8s.io/api v0.17.2

	// k8s.io/apimachinery 1.16.3 is at 72ed19daf4bb
	k8s.io/apimachinery v0.17.2

	// k8s.io/client-go 1.16.3 is at 6c5935290e33
	k8s.io/client-go v0.17.2
)

replace (
	github.com/bronze1man/goStrongswanVici => github.com/tigera/goStrongswanVici v0.0.0-20180704141420-9b6fdd821dbe
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200303175443-c7538b62e113
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.0.0-20200304163605-aadd9f396ddc
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	github.com/vishvananda/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320
)
