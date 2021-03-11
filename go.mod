module github.com/tigera/es-proxy

go 1.15

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20210309223655-a5a1871dce03
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210311183155-746326c47108
	// Need to pin typha to get go mod updates for felix to go through.
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20210311184453-ac0967ab6e10
	github.com/tigera/apiserver => github.com/tigera/apiserver v0.0.0-20210305163939-5088de0fbab0
	github.com/tigera/compliance => github.com/tigera/compliance v0.0.0-20210311191826-40eb13f53bb2
	github.com/tigera/lma => github.com/tigera/lma v0.0.0-20210311183729-fcbc7dbc0b30

	k8s.io/api => k8s.io/api v0.19.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.6
	// Using cloned tigera/apimachinery-private cloned off k8s apimachinery kubernetes 1.19.6
	k8s.io/apimachinery => github.com/tigera/apimachinery-private v0.0.0-20210113215858-25fe336c7928
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

require (
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/google/go-cmp v0.5.2
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/olivere/elastic/v7 v7.0.22
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	github.com/tigera/apiserver v2.7.0-0.dev.0.20200106212250-74a03f23227a+incompatible
	github.com/tigera/compliance v0.0.0-20201124233520-d4b5ad65a5a6
	github.com/tigera/lma v0.0.0-20210311183729-fcbc7dbc0b30
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	k8s.io/api v0.19.6
	k8s.io/apimachinery v0.19.6
	k8s.io/apiserver v0.19.6
	k8s.io/client-go v10.0.0+incompatible
)
