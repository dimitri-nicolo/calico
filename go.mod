module github.com/tigera/calicoq

go 1.15

require (
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/go-ini/ini v1.49.0 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/projectcalico/felix v3.8.9+incompatible
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/sirupsen/logrus v1.7.0
	github.com/tigera/licensing v1.0.1-0.20210422165141-a2325849af0e
)

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20210423191756-c6173f117a07
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210422164824-326b0db88236
	// Need to pin typha to get go mod updates for felix to go through.
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20210422170328-e25ccb20e002
	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.4.2-0.20190403091019-9b3cdde74fbe

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
