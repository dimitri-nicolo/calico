module github.com/tigera/packetcapture-api

go 1.15

require (
	github.com/emicklei/go-restful v2.11.2-0.20200112161605-a7c079c43d51+incompatible // indirect
	github.com/go-openapi/jsonreference v0.19.4-0.20191224164422-1f9748e5f45e // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/projectcalico/apiserver v0.0.0-20210423155446-68f31180801c
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/tigera/lma v0.0.0-20210709012013-a1255726e573
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/apiserver v0.21.0
	k8s.io/client-go v10.0.0+incompatible
)

replace (
	github.com/projectcalico/apiserver => github.com/tigera/apiserver v0.0.0-20210709011444-86b1ba7b9f55 // indirect
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210709010357-9cc9c9f2e861

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
	k8s.io/node-api => k8s.io/node-api v0.21.0-rc.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.0-rc.0
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.21.0-rc.0
	k8s.io/sample-controller => k8s.io/sample-controller v0.21.0-rc.0
)
