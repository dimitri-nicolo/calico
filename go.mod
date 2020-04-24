module github.com/projectcalico/kube-controllers

go 1.13

require (
	github.com/apparentlymart/go-cidr v1.0.1

	// Elastic repo for ECK v1.0.1
	github.com/elastic/cloud-on-k8s v0.0.0-20200204083752-bcb7468838a8
	github.com/elastic/go-elasticsearch/v7 v7.3.0
	github.com/joho/godotenv v1.3.0
	github.com/kelseyhightower/envconfig v1.4.0

	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.1
	github.com/patrickmn/go-cache v0.0.0-20180815053127-5633e0862627
	github.com/projectcalico/felix v0.0.0-00010101000000-000000000000
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/pflag v1.0.5
	github.com/tigera/api v0.0.0-20200115221514-2e8e59c327b0
	github.com/tigera/licensing v1.0.1-0.20200417212345-02da246de3e1
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738

	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/apiserver v0.17.3
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/klog v1.0.0
)

replace (
	github.com/projectcalico/cni-plugin => github.com/tigera/cni-plugin-private v0.0.0-20200423042032-b173c8e671c7
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20200422041113-2b9f3dbe5ff3
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200423003215-56e38b498404
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.0.0-20200423100815-abc739365bad
	// We need to hold back prometheus/client_golang to avoid a build failure. This is hopefully a
	// temporary fix.
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.4
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	github.com/vishvananda/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320

	k8s.io/api => k8s.io/api v0.17.2

	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.2
	// Using cloned tigera/apimachinery-private cloned off k8s apimachinery kubernetes 1.17.2
	k8s.io/apimachinery => github.com/tigera/apimachinery-private v0.0.0-20200210212631-f989df51e340
	k8s.io/apiserver => k8s.io/apiserver v0.17.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.2
	k8s.io/client-go => k8s.io/client-go v0.17.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.2
	k8s.io/code-generator => k8s.io/code-generator v0.17.2
	k8s.io/component-base => k8s.io/component-base v0.17.2
	k8s.io/cri-api => k8s.io/cri-api v0.17.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.2
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.2
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.2
	k8s.io/kubectl => k8s.io/kubectl v0.17.2
	k8s.io/kubelet => k8s.io/kubelet v0.17.2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.2
	k8s.io/metrics => k8s.io/metrics v0.17.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.2
)
