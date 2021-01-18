module github.com/tigera/es-proxy

go 1.15

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20210115194353-681da7eec076
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210108223937-a3b320c37bbb
	// Need to pin typha to get go mod updates for felix to go through.
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20210108235021-216cabe9d0bd
	github.com/tigera/apiserver => github.com/tigera/apiserver v0.0.0-20210118191038-5e4733c5a4f4
	github.com/tigera/compliance => github.com/tigera/compliance v0.0.0-20210114181251-b1079a3a142a
	github.com/tigera/lma => github.com/tigera/lma v0.0.0-20210118191616-4b5668956085

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
	github.com/google/go-cmp v0.4.0
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/olivere/elastic/v7 v7.0.6
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	github.com/tigera/apiserver v2.7.0-0.dev.0.20200106212250-74a03f23227a+incompatible
	github.com/tigera/compliance v0.0.0-20201124233520-d4b5ad65a5a6
	github.com/tigera/lma v0.0.0-20210114175836-5b76fea7ad37
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	k8s.io/api v0.19.6
	k8s.io/apimachinery v0.19.6
	k8s.io/apiserver v0.19.6
	k8s.io/client-go v10.0.0+incompatible
)
