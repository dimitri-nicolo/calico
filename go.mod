module github.com/tigera/es-proxy

go 1.16

require (
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/google/go-cmp v0.5.6
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/olivere/elastic/v7 v7.0.22
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.6.1
	github.com/tigera/api v0.0.0-20211202170222-d8128d06db71
	github.com/tigera/compliance v0.0.0-20211207130538-788b9e2b6d0c
	github.com/tigera/lma v0.0.0-20211202211719-6d7ab97191be
	k8s.io/api v0.21.7
	k8s.io/apimachinery v0.21.7
	k8s.io/apiserver v0.21.7
	k8s.io/client-go v10.0.0+incompatible
)

replace (
	github.com/projectcalico/apiserver => github.com/tigera/apiserver v0.0.0-20211202200625-2402c45cf93e
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20211207125308-bd3de6cfc0e0
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20211203232041-873deb396265
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20211202200643-d13e06a0845c

	k8s.io/api => k8s.io/api v0.21.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.7
	k8s.io/apiserver => k8s.io/apiserver v0.21.7
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.7
	k8s.io/client-go => k8s.io/client-go v0.21.7
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.7
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.7
	k8s.io/code-generator => k8s.io/code-generator v0.21.7
	k8s.io/component-base => k8s.io/component-base v0.21.7
	k8s.io/component-helpers => k8s.io/component-helpers v0.21.7
	k8s.io/controller-manager => k8s.io/controller-manager v0.21.7
	k8s.io/cri-api => k8s.io/cri-api v0.21.7
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.7
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.7
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.7
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20211110012726-3cc51fd1e909
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.7
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.7
	k8s.io/kubectl => k8s.io/kubectl v0.21.7
	k8s.io/kubelet => k8s.io/kubelet v0.21.7
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.7
	k8s.io/metrics => k8s.io/metrics v0.21.7
	k8s.io/mount-utils => k8s.io/mount-utils v0.21.7
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.7
)
