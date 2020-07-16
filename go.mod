module github.com/tigera/apiserver

go 1.13

require (
	github.com/go-openapi/spec v0.19.4
	github.com/google/gofuzz v1.0.0
	github.com/onsi/gomega v1.7.1
	github.com/pkg/errors v0.8.1
	github.com/projectcalico/libcalico-go v1.7.2
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/tigera/licensing v1.0.1-0.20200617233549-fa90126a1814
	golang.org/x/net v0.0.0-20200202094626-16171245cfb2
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/apiserver v0.17.3
	k8s.io/client-go v0.17.3
	k8s.io/code-generator v0.17.3
	k8s.io/component-base v0.17.3
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/kubernetes v1.17.3
)

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20200716205919-ddcc11d84eb6
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2

	k8s.io/api => k8s.io/api v0.17.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.3

	// Using cloned tigera/apimachinery-private cloned off k8s apimachinery kubernetes 1.17.3
	k8s.io/apimachinery => github.com/tigera/apimachinery-private v0.0.0-20200406201717-c612ebea4f6b

	k8s.io/apiserver => k8s.io/apiserver v0.17.3

	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.3
	k8s.io/client-go => k8s.io/client-go v0.17.3

	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.3

	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.3

	k8s.io/code-generator => k8s.io/code-generator v0.17.3
	k8s.io/component-base => k8s.io/component-base v0.17.3
	k8s.io/cri-api => k8s.io/cri-api v0.17.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.3
	k8s.io/kubectl => k8s.io/kubectl v0.17.3
	k8s.io/kubelet => k8s.io/kubelet v0.17.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.3
	k8s.io/metrics => k8s.io/metrics v0.17.3
	k8s.io/node-api => k8s.io/node-api v0.17.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.3
)
