module github.com/tigera/es-proxy

go 1.13

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20200403234312-901077b0ebea
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20200403193804-6ff5076fc1c8
	github.com/tigera/apiserver => github.com/tigera/apiserver v0.0.0-20200406043105-c5e752dc8e4a
	github.com/tigera/compliance => github.com/tigera/compliance v0.0.0-20200404083203-fdd38cb411be
	github.com/tigera/lma => github.com/tigera/lma v0.0.0-20200404190459-ae48a942fddf

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

require (
	github.com/cloudflare/cfssl v0.0.0-20180726162950-56268a613adf // indirect
	github.com/coreos/etcd v3.3.17+incompatible // indirect
	github.com/coreos/rkt v1.30.0 // indirect
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/godbus/dbus v4.1.0+incompatible // indirect
	github.com/google/certificate-transparency-go v1.0.21 // indirect
	github.com/heketi/rest v0.0.0-20180404230133-aa6a65207413 // indirect
	github.com/heketi/utils v0.0.0-20170317161834-435bc5bdfa64 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/olivere/elastic/v7 v7.0.6
	github.com/onsi/ginkgo v1.10.2
	github.com/onsi/gomega v1.7.1
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/sirupsen/logrus v1.4.2
	github.com/tigera/apiserver v2.7.0-0.dev.0.20200106212250-74a03f23227a+incompatible
	github.com/tigera/compliance v0.0.0-20200321083204-39304111550f
	github.com/tigera/lma v0.0.0-20200321212149-273f27b4c88b
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v11.0.0+incompatible
)
