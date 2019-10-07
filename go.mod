module github.com/projectcalico/felix

go 1.12

require (
	github.com/Microsoft/go-winio v0.0.0-20190408173621-84b4ab48a507 // indirect
	github.com/Microsoft/hcsshim v0.0.0-20190408221605-063ae4a83d78
	github.com/aws/aws-sdk-go v1.13.54
	github.com/bronze1man/goStrongswanVici v0.0.0-20190828090544-27d02f80ba40
	github.com/containernetworking/cni v0.5.2
	github.com/coreos/etcd v3.3.10+incompatible // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/gavv/monotime v0.0.0-20190418164738-30dba4353424
	github.com/go-ini/ini v0.0.0-20190327024845-3be5ad479f69
	github.com/gobuffalo/packr/v2 v2.0.9
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/golang/protobuf v1.3.2
	github.com/google/btree v1.0.0 // indirect
	github.com/google/gopacket v0.0.0-20190313190028-b7586607157b
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gophercloud/gophercloud v0.4.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/gxed/GoEndian v0.0.0-20160916112711-0f5c6873267e // indirect
	github.com/gxed/eventfd v0.0.0-20160916113412-80a92cca79a8 // indirect
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/ipfs/go-log v0.0.0-20180611222144-5dc2060baaf8 // indirect
	github.com/jmespath/go-jmespath v0.0.0-20151117175822-3433f3ea46d9 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/kelseyhightower/envconfig v0.0.0-20180517194557-dd1402a4d99d
	github.com/libp2p/go-reuseport v0.0.0-20180924121034-dd0c37d7767b
	github.com/libp2p/go-sockaddr v0.0.0-20190411201116-52957a0228cc // indirect
	github.com/mattn/go-colorable v0.1.1 // indirect
	github.com/mattn/go-isatty v0.0.7 // indirect
	github.com/mipearson/rfw v0.0.0-20170619235010-6f0a6f3266ba
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/projectcalico/pod2daemon v3.7.5+incompatible
	github.com/projectcalico/typha v3.8.2+incompatible
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910
	github.com/rogpeppe/go-internal v1.3.0 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/tigera/licensing v2.6.0-0.dev+incompatible
	github.com/tigera/nfnetlink v0.0.0-20190401090543-2623d65568be
	github.com/ugorji/go v1.1.7 // indirect
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20160430053723-8ba1072b58e0 // indirect
	github.com/whyrusleeping/go-logging v0.0.0-20170515211332-0457bb6b88fc // indirect
	golang.org/x/crypto v0.0.0-20190911031432-227b76d455e7 // indirect
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc
	golang.org/x/sys v0.0.0-20190616124812-15dcb6c0061f
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/grpc v1.19.0
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0-20170531160350-a96e63847dc3
	gopkg.in/square/go-jose.v2 v2.0.0-20180411045311-89060dee6a84 // indirect
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/client-go v8.0.0+incompatible
)

replace (
	github.com/bronze1man/goStrongswanVici => github.com/tigera/goStrongswanVici v0.0.0-20180704141420-9b6fdd821dbe
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20191006231700-5cc9448298e3
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.0.0-20191007163426-b37d91ed61f4
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	github.com/vishvananda/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320
)
