module github.com/tigera/honeypod-controller

go 1.15

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20210212183343-ca6b53813ee3
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210212223417-27c85309f19e
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20210212224648-d1636500d28f

	k8s.io/api => k8s.io/api v0.19.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.6
	// Using tigera/apimachinery-private cloned off k8s apimachinery kubernetes 1.18.12
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
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/sirupsen/logrus v1.6.0
	github.com/tigera/licensing v1.0.1-0.20210208225242-c586047f6f54
	github.com/tigera/lma v0.0.0-20210212223922-928821b1f267
	k8s.io/client-go v11.0.0+incompatible // indirect
)
