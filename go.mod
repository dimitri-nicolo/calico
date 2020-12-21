module github.com/tigera/apiserver

go 1.15

require (
	github.com/go-openapi/spec v0.19.4
	github.com/google/gofuzz v1.1.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/pkg/errors v0.8.1
	github.com/projectcalico/libcalico-go v1.7.2
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	github.com/tigera/licensing v1.0.1-0.20201218173202-d766f2323fee
	golang.org/x/net v0.0.0-20200520004742-59133d7f0dd7
	k8s.io/api v0.18.12
	k8s.io/apimachinery v0.18.12
	k8s.io/apiserver v0.18.12
	k8s.io/client-go v0.18.12
	k8s.io/code-generator v0.18.12
	k8s.io/component-base v0.18.12
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
	k8s.io/kubernetes v1.18.12
)

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20201215061722-3d90f1306e73
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2

	k8s.io/api => k8s.io/api v0.18.12
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.12

	// Using cloned tigera/apimachinery-private cloned off k8s apimachinery kubernetes 1.18.12
	k8s.io/apimachinery => github.com/tigera/apimachinery-private v0.0.0-20201204234441-e565126b30e8

	k8s.io/apiserver => k8s.io/apiserver v0.18.12

	k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.12
	k8s.io/client-go => k8s.io/client-go v0.18.12

	k8s.io/cloud-provider => k8s.io/cloud-provider v0.18.12

	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.18.12

	k8s.io/code-generator => k8s.io/code-generator v0.18.12
	k8s.io/component-base => k8s.io/component-base v0.18.12
	k8s.io/cri-api => k8s.io/cri-api v0.18.12
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.18.12
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.12
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.18.12
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.18.12
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.18.12
	k8s.io/kubectl => k8s.io/kubectl v0.18.12
	k8s.io/kubelet => k8s.io/kubelet v0.18.12
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.18.12
	k8s.io/metrics => k8s.io/metrics v0.18.12
	k8s.io/node-api => k8s.io/node-api v0.18.12
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.18.12
)
