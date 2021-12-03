module github.com/projectcalico/felix

go 1.16

require (
	github.com/Microsoft/hcsshim v0.8.10-0.20200715222032-5eafd1556990
	github.com/aws/aws-sdk-go-v2 v1.7.1
	github.com/aws/aws-sdk-go-v2/config v1.5.0
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.3.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.12.0
	github.com/aws/smithy-go v1.6.0
	github.com/bronze1man/goStrongswanVici v0.0.0-20190828090544-27d02f80ba40
	github.com/containernetworking/plugins v0.8.5
	github.com/davecgh/go-spew v1.1.1
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/fastly/go-utils v0.0.0-20180712184237-d95a45783239 // indirect
	github.com/florianl/go-nfqueue v1.2.0
	github.com/gavv/monotime v0.0.0-20190418164738-30dba4353424
	github.com/go-ini/ini v1.44.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.6
	github.com/google/gopacket v1.1.19
	github.com/google/netstack v0.0.0-20191123085552-55fcc16cd0eb
	github.com/google/uuid v1.2.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/ishidawataru/sctp v0.0.0-20191218070446-00ab2ac2db07
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.3 // indirect
	github.com/libp2p/go-reuseport v0.0.1
	github.com/mdlayher/netlink v1.4.1
	github.com/mipearson/rfw v0.0.0-20170619235010-6f0a6f3266ba
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/pkg/errors v0.9.1
	github.com/projectcalico/libcalico-go v1.7.2
	github.com/projectcalico/pod2daemon v0.0.0-20211020230039-cbd1482bd13b
	github.com/projectcalico/typha v0.7.3-0.20210428181500-9e435d5fd964
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.26.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/tebeka/strftime v0.1.5 // indirect
	github.com/tigera/api v0.0.0-20211130232353-e7a188095a30
	github.com/tigera/licensing v1.0.1-0.20211120000321-4024ba82fe25
	github.com/tigera/nfnetlink v0.0.0-20210819183736-75abca8ede69
	github.com/tigera/windows-networking v0.0.0-20211112174220-6a90051af748
	github.com/vishvananda/netlink v1.1.1-0.20210703095558-21f2c55a7727
	github.com/willf/bitset v1.1.11
	golang.org/x/net v0.0.0-20210927181540-4e4d966f7476
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210927094055-39ccf1dd6fa6
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20211027115401-c9b1ec1aa6d8
	google.golang.org/grpc v1.27.1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/tchap/go-patricia.v2 v2.3.0
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/apiserver v0.22.0 // indirect
	k8s.io/client-go v0.22.0
	k8s.io/kubernetes v1.21.0
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176
	modernc.org/memory v1.0.4
	sigs.k8s.io/kind v0.11.1
)

replace (
	github.com/Microsoft/hcsshim => github.com/projectcalico/hcsshim v0.8.9-calico
	github.com/bronze1man/goStrongswanVici => github.com/tigera/goStrongswanVici v0.0.0-20180704141420-9b6fdd821dbe
	github.com/projectcalico/libcalico-go => github.com/fasaxc/libcalico-go-private v1.7.2-0.20211203114103-460e7a875707
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20211119195348-94cd571f25b8
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico

	// The GRPC library seems to float to latest without this (not sure why!) and v1.30+ are incompatible with
	// the etcd client.
	google.golang.org/grpc => google.golang.org/grpc v1.29.1

	// Need replacements for all the k8s subsidiary projects that are pulled in indirectly because
	// the Kubernetes repo pulls them in via a replacement to its own vendored copies, which doesn't work for
	// transient imports.
	k8s.io/api => k8s.io/api v0.21.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.0
	k8s.io/apiserver => k8s.io/apiserver v0.21.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.0
	k8s.io/client-go => k8s.io/client-go v0.21.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.0
	k8s.io/code-generator => k8s.io/code-generator v0.21.0
	k8s.io/component-base => k8s.io/component-base v0.21.0
	k8s.io/component-helpers => k8s.io/component-helpers v0.21.0
	k8s.io/controller-manager => k8s.io/controller-manager v0.21.0
	k8s.io/cri-api => k8s.io/cri-api v0.21.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.0
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.0
	k8s.io/kubectl => k8s.io/kubectl v0.21.0
	k8s.io/kubelet => k8s.io/kubelet v0.21.0
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.0
	k8s.io/metrics => k8s.io/metrics v0.21.0
	k8s.io/mount-utils => k8s.io/mount-utils v0.21.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.0
)
