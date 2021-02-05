module github.com/projectcalico/kube-controllers

go 1.15

require (
	github.com/apparentlymart/go-cidr v1.0.1

	// Elastic repo for ECK v1.0.1
	github.com/elastic/cloud-on-k8s v0.0.0-20200204083752-bcb7468838a8
	github.com/elastic/go-elasticsearch/v7 v7.3.0
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/projectcalico/felix v3.8.9+incompatible
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/projectcalico/typha v0.7.3-0.20201007232318-2dba00d728ad
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.5.1
	github.com/tigera/api v0.0.0-20201210003641-f72711afec9e
	github.com/tigera/licensing v1.0.1-0.20210123223002-53d994486b81
	go.etcd.io/etcd v0.5.0-alpha.5.0.20201125193152-8a03d2e9614b
	honnef.co/go/tools v0.0.1-2020.1.4 // indirect
	k8s.io/api v0.19.6
	k8s.io/apimachinery v0.19.6
	k8s.io/apiserver v0.19.6
	k8s.io/client-go v0.19.6
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.7.0 // indirect
)

replace (
	github.com/projectcalico/cni-plugin => github.com/tigera/cni-plugin-private v1.11.1-0.20210129214514-9beed9cb19fc
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20210205124834-2978cde517b7
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210126182422-b334ce1a1d94
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20210127030435-99c41f61b97f

	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	github.com/vishvananda/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320

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
