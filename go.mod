module github.com/projectcalico/felix

go 1.15

require (
	github.com/Microsoft/hcsshim v0.8.10-0.20200715222032-5eafd1556990
	github.com/aws/aws-sdk-go v1.35.7
	github.com/bronze1man/goStrongswanVici v0.0.0-20190828090544-27d02f80ba40
	github.com/containernetworking/plugins v0.8.5
	github.com/davecgh/go-spew v1.1.1
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/fastly/go-utils v0.0.0-20180712184237-d95a45783239 // indirect
	github.com/gavv/monotime v0.0.0-20190418164738-30dba4353424
	github.com/go-ini/ini v1.44.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.4.2
	github.com/google/gopacket v1.1.18
	github.com/google/netstack v0.0.0-20191123085552-55fcc16cd0eb
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/ishidawataru/sctp v0.0.0-20191218070446-00ab2ac2db07
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.3 // indirect
	github.com/libp2p/go-reuseport v0.0.1
	github.com/mipearson/rfw v0.0.0-20170619235010-6f0a6f3266ba
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/pkg/errors v0.9.1
	github.com/projectcalico/libcalico-go v1.7.2
	github.com/projectcalico/pod2daemon v0.0.0-20210303174516-f4d124e8d7e6
	github.com/projectcalico/typha v0.7.3-0.20201007232318-2dba00d728ad
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.6.1
	github.com/stretchr/testify v1.5.1
	github.com/tebeka/strftime v0.1.5 // indirect
	github.com/tigera/licensing v1.0.1-0.20210329174156-fbcaf2a3b8aa
	github.com/tigera/nfnetlink v0.0.0-20210105181106-4ac282dc9ec8
	github.com/tigera/windows-networking v0.0.0-20210216141047-0f2491c3b4e1
	github.com/vishvananda/netlink v1.1.0
	github.com/willf/bitset v1.1.11
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/sys v0.0.0-20201112073958-5cba982894dd
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20200324154536-ceff61240acf
	google.golang.org/grpc v1.27.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/tchap/go-patricia.v2 v2.3.0
	// The matching kubernetes subsidiary projects have matching tags that are one major version behind the main repo.
	k8s.io/api v0.19.6
	k8s.io/apimachinery v0.19.6
	k8s.io/client-go v0.19.6

	// Felix imports kubernetes itself to pick up the kube-proxy business logic.
	k8s.io/kubernetes v1.18.12
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73
	sigs.k8s.io/kind v0.9.0
)

replace (
	github.com/Microsoft/hcsshim => github.com/projectcalico/hcsshim v0.8.9-calico
	github.com/bronze1man/goStrongswanVici => github.com/tigera/goStrongswanVici v0.0.0-20180704141420-9b6fdd821dbe
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210330193744-b0b306b0c9bc
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20210330195141-36c677e4ab3c
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	github.com/vishvananda/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320

	// Need replacements for all the k8s subsidiary projects that are pulled in indirectly because
	// the kubernets repo pulls them in via a replacement to its own vendored copies, which doesn't work for
	// transient imports.
	k8s.io/api => k8s.io/api v0.19.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.6
	k8s.io/apiserver => k8s.io/apiserver v0.19.6
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.6
	k8s.io/client-go => k8s.io/client-go v0.19.6
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.6
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.6
	k8s.io/code-generator => k8s.io/code-generator v0.19.6
	k8s.io/component-base => k8s.io/component-base v0.19.6
	k8s.io/cri-api => k8s.io/cri-api v0.19.6
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.6
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.6
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.6
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.6
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.6
	k8s.io/kubectl => k8s.io/kubectl v0.19.6
	k8s.io/kubelet => k8s.io/kubelet v0.19.6
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.6
	k8s.io/metrics => k8s.io/metrics v0.19.6
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.6
)
