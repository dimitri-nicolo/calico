module github.com/tigera/es-proxy

go 1.15

require (
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/google/go-cmp v0.5.4
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/olivere/elastic/v7 v7.0.22
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/projectcalico/apiserver v0.0.0-20210423155446-68f31180801c
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/tigera/compliance v0.0.0-20210511230907-24324abdd665
	github.com/tigera/lma v0.0.0-20210511170025-64ef8d158b76
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/apiserver v0.21.0
	k8s.io/client-go v11.0.0+incompatible
)

replace (
	github.com/projectcalico/apiserver => github.com/tigera/apiserver v0.0.0-20210514222227-aadcfc544a5c
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20210511195521-99eb57ea3ef9
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210511163730-078ad1161770
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20210513005647-366239b8ed7a

	k8s.io/api => k8s.io/api v0.21.0-rc.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.0-rc.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.0-rc.0
	k8s.io/apiserver => k8s.io/apiserver v0.21.0-rc.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.0-rc.0
	k8s.io/client-go => k8s.io/client-go v0.21.0-rc.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.0-rc.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.0-rc.0
	k8s.io/code-generator => k8s.io/code-generator v0.21.0-rc.0
	k8s.io/component-base => k8s.io/component-base v0.21.0-rc.0
	k8s.io/component-helpers => k8s.io/component-helpers v0.21.0-rc.0
	k8s.io/controller-manager => k8s.io/controller-manager v0.21.0-rc.0
	k8s.io/cri-api => k8s.io/cri-api v0.21.0-rc.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.0-rc.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.0-rc.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.0-rc.0
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.0-rc.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.0-rc.0
	k8s.io/kubectl => k8s.io/kubectl v0.21.0-rc.0
	k8s.io/kubelet => k8s.io/kubelet v0.21.0-rc.0
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.0-rc.0
	k8s.io/metrics => k8s.io/metrics v0.21.0-rc.0
	k8s.io/mount-utils => k8s.io/mount-utils v0.21.0-rc.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.0-rc.0
)
