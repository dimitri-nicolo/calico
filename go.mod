module github.com/tigera/prometheus-service

go 1.16

require (
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/projectcalico/apiserver v0.0.0-20210423155446-68f31180801c // indirect
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/common v0.29.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tigera/lma v0.0.0-20211202211719-6d7ab97191be
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	golang.org/x/sys v0.0.0-20210616094352-59db8d763f22 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	k8s.io/api v0.21.7
	k8s.io/apimachinery v0.21.7
	k8s.io/apiserver v0.21.7
	k8s.io/client-go v0.21.7
)

replace (
	github.com/projectcalico/apiserver => github.com/tigera/apiserver v0.0.0-20211202200625-2402c45cf93e
	github.com/projectcalico/libcalico-go => github.com/projectcalico/libcalico-go v1.7.2-0.20211201221416-38b1671dbe42

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
	k8s.io/node-api => k8s.io/node-api v0.21.7
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.7
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.21.7
	k8s.io/sample-controller => k8s.io/sample-controller v0.21.7
)
