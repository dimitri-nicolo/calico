module github.com/tigera/honeypod-recommendation

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20200617214044-dd9aed324aa2
	//github.com/tigera/lma => github.com/tigera/lma v0.0.0-20200826003057-05f425c79e8c
	//github.com/tigera/lma => github.com/tigera/lma v0.0.0-20200909031343-e24ceeb5cc73
	github.com/tigera/lma => github.com/tigera/lma v0.0.0-20200909161544-33181291211a
	k8s.io/api => k8s.io/api v0.17.2

	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.2
	// Using cloned tigera/apimachinery-private cloned off k8s apimachinery kubernetes 1.17.2
	k8s.io/apimachinery => github.com/tigera/apimachinery-private v0.0.0-20200210212631-f989df51e340
	k8s.io/apiserver => k8s.io/apiserver v0.17.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.2
	k8s.io/client-go => k8s.io/client-go v0.17.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.2
	k8s.io/code-generator => k8s.io/code-generator v0.17.2
	k8s.io/component-base => k8s.io/component-base v0.17.2
	k8s.io/cri-api => k8s.io/cri-api v0.17.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.2
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.2
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.2
	k8s.io/kubectl => k8s.io/kubectl v0.17.2
	k8s.io/kubelet => k8s.io/kubelet v0.17.2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.2
	k8s.io/metrics => k8s.io/metrics v0.17.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.2
)

require (
	github.com/projectcalico/libcalico-go v1.7.3 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/tigera/lma v0.0.0-20200909215637-e214c5a9cc07
	gopkg.in/yaml.v2 v2.3.0 // indirect
	k8s.io/client-go v11.0.0+incompatible // indirect
	k8s.io/klog v1.0.0
)

go 1.14
