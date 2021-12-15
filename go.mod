module github.com/projectcalico/calico

go 1.16

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Masterminds/sprig v2.20.0+incompatible
	github.com/Microsoft/hcsshim v0.8.10-0.20200715222032-5eafd1556990
	github.com/StackExchange/wmi v0.0.0-20181212234831-e0a55b97c705 // indirect
	github.com/Workiva/go-datastructures v1.0.53
	github.com/alecthomas/participle v0.7.1
	github.com/apparentlymart/go-cidr v1.1.0
	github.com/aws/aws-sdk-go-v2 v1.11.0
	github.com/aws/aws-sdk-go-v2/config v1.10.0
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.8.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.21.0
	github.com/aws/smithy-go v1.9.0
	github.com/bronze1man/goStrongswanVici v0.0.0-20190828090544-27d02f80ba40
	github.com/buger/jsonparser v1.0.0
	github.com/containernetworking/cni v0.8.0
	github.com/containernetworking/plugins v0.8.5
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/distribution/distribution v2.7.1+incompatible
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/eapache/channels v1.1.0 // indirect
	github.com/elastic/cloud-on-k8s v0.0.0-20200204083752-bcb7468838a8
	github.com/elastic/go-elasticsearch/v7 v7.3.0
	github.com/envoyproxy/go-control-plane v0.9.8
	github.com/fastly/go-utils v0.0.0-20180712184237-d95a45783239 // indirect
	github.com/florianl/go-nfqueue v1.2.0
	github.com/gavv/monotime v0.0.0-20190418164738-30dba4353424
	github.com/ghodss/yaml v1.0.0
	github.com/go-ini/ini v1.63.2
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/spec v0.19.5
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/go-playground/locales v0.12.1 // indirect
	github.com/go-playground/universal-translator v0.0.0-20170327191703-71201497bace // indirect
	github.com/go-sql-driver/mysql v1.4.0
	github.com/gofrs/flock v0.8.0
	github.com/gogo/googleapis v1.4.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.6
	github.com/google/gopacket v1.1.19
	github.com/google/netstack v0.0.0-20191123085552-55fcc16cd0eb
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/howeyc/fsnotify v0.9.0
	github.com/influxdata/influxdb v0.0.0-20190102202943-dd481f35df2c // indirect
	github.com/influxdata/platform v0.0.0-20190117200541-d500d3cf5589 // indirect
	github.com/ishidawataru/sctp v0.0.0-20191218070446-00ab2ac2db07
	github.com/jehiah/go-strftime v0.0.0-20171201141054-1d33003b3869 // indirect
	github.com/joho/godotenv v1.4.0
	github.com/juju/clock v0.0.0-20190205081909-9c5c9712527c
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/mutex v0.0.0-20180619145857-d21b13acf4bf
	github.com/juju/testing v0.0.0-20200608005635-e4eedbc6f7aa // indirect
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.1.0
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kelseyhightower/memkv v0.1.1
	github.com/leodido/go-urn v0.0.0-20181204092800-a67a23e1c1af // indirect
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.3 // indirect
	github.com/libp2p/go-reuseport v0.1.0
	github.com/mcuadros/go-version v0.0.0-20190308113854-92cdf37c5b75
	github.com/mdlayher/netlink v1.4.1
	github.com/mipearson/rfw v0.0.0-20170619235010-6f0a6f3266ba
	github.com/mitchellh/go-homedir v1.1.0
	github.com/natefinch/atomic v0.0.0-20150920032501-a62ce929ffcc
	github.com/nmrshll/go-cp v0.0.0-20180115193924-61436d3b7cfa
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/osrg/gobgp v0.0.0-20170802061517-bbd1d99396fe
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.26.0
	github.com/rakelkar/gonetsh v0.3.2
	github.com/robfig/cron v1.1.0
	github.com/satori/go.uuid v1.2.0
	github.com/shirou/gopsutil v0.0.0-20190323131628-2cbc9195c892
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.0
	github.com/tchap/go-patricia/v2 v2.3.1
	github.com/tebeka/strftime v0.1.5 // indirect
	github.com/termie/go-shutil v0.0.0-20140729215957-bcacb06fecae
	github.com/tigera/api v0.0.0-20211119192830-60ae1a27d9ca
	github.com/tigera/windows-networking v0.0.0-20211112174220-6a90051af748
	github.com/vishvananda/netlink v1.1.1-0.20210703095558-21f2c55a7727
	github.com/willf/bitset v1.1.11
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0
	go.etcd.io/etcd v0.5.0-alpha.5.0.20201125193152-8a03d2e9614b
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
	golang.org/x/net v0.0.0-20210927181540-4e4d966f7476
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210927094055-39ccf1dd6fa6
	golang.zx2c4.com/wireguard v0.0.20200121 // indirect
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20211027115401-c9b1ec1aa6d8
	google.golang.org/genproto v0.0.0-20210828152312-66f60bf46e71
	google.golang.org/grpc v1.41.0
	gopkg.in/fsnotify/fsnotify.v1 v1.4.7
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	// validator.v9 must be at v9.30.2
	gopkg.in/go-playground/validator.v9 v9.30.2
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/tchap/go-patricia.v2 v2.3.0
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.4
	k8s.io/apiextensions-apiserver v0.22.2
	k8s.io/apimachinery v0.22.4
	k8s.io/apiserver v0.21.8-rc.0
	k8s.io/client-go v0.22.4
	k8s.io/code-generator v0.21.0
	k8s.io/component-base v0.22.2
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20211110012726-3cc51fd1e909
	k8s.io/kubernetes v1.21.0
	k8s.io/utils v0.0.0-20210819203725-bdf08cb9a70a
	modernc.org/memory v1.0.5
	sigs.k8s.io/controller-runtime v0.9.0-alpha.1 // indirect
	sigs.k8s.io/kind v0.11.1
)

replace (
	github.com/Microsoft/hcsshim => github.com/projectcalico/hcsshim v0.8.9-calico
	github.com/bronze1man/goStrongswanVici => github.com/tigera/goStrongswanVici v0.0.0-20180704141420-9b6fdd821dbe
	github.com/containernetworking/cni => github.com/containernetworking/cni v0.8.0

	// Replace the envoy data-plane-api dependency with the projectcalico fork that includes the generated
	// go bindings for the API. Upstream only includes the protobuf definitions, so we need to fork in order to
	// supply the go code.
	github.com/envoyproxy/data-plane-api => github.com/projectcalico/data-plane-api v0.0.0-20210121211707-a620ff3c8f7e

	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	github.com/tigera/api => ./api
	google.golang.org/grpc => google.golang.org/grpc v1.29.1
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

	// Need replacements for all the k8s subsidiary projects that are pulled in indirectly because
	// the kubernets repo pulls them in via a replacement to its own vendored copies, which doesn't work for
	// transient imports.
	k8s.io/kubernetes => k8s.io/kubernetes v1.21.0
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.0
	k8s.io/metrics => k8s.io/metrics v0.21.0
	k8s.io/mount-utils => k8s.io/mount-utils v0.21.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.0
)
