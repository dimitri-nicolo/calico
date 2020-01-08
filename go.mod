module github.com/tigera/calico-k8sapiserver

go 1.12

require (
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/go-openapi/spec v0.19.3
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/google/gofuzz v1.0.0
	github.com/huandu/xstrings v1.2.1 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	//github.com/kubernetes/repo-infra v0.0.0-20191204005258-28d05af9a236
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/projectcalico/libcalico-go v0.0.0-00010101000000-000000000000
	github.com/prometheus/client_golang v1.3.0 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/tigera/licensing v0.0.0-20191114203016-3c126d3f9ffe
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9
	golang.org/x/tools v0.0.0-20191211233649-dfebd2802ad3 // indirect
	k8s.io/api v0.16.4
	k8s.io/apiextensions-apiserver v0.16.4 // indirect
	k8s.io/apimachinery v0.16.4
	k8s.io/apiserver v0.16.4
	k8s.io/client-go v0.16.4
	k8s.io/code-generator v0.16.4
	k8s.io/component-base v0.16.4
	k8s.io/gengo v0.0.0-20191120174120-e74f70b9b27e // indirect
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/kubernetes v1.16.4
)

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200102171614-37ba08c32fe9

	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2

	k8s.io/api => k8s.io/api v0.16.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.16.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.4
	k8s.io/apiserver => k8s.io/apiserver v0.16.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.16.4
	k8s.io/client-go => k8s.io/client-go v0.16.4

	k8s.io/cloud-provider => k8s.io/cloud-provider v0.16.4

	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.16.4

	k8s.io/code-generator => k8s.io/code-generator v0.16.4
	k8s.io/component-base => k8s.io/component-base v0.16.4
	k8s.io/cri-api => k8s.io/cri-api v0.16.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.16.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.16.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.16.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.16.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.16.4
	k8s.io/kubectl => k8s.io/kubectl v0.16.4
	k8s.io/kubelet => k8s.io/kubelet v0.16.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.16.4
	k8s.io/metrics => k8s.io/metrics v0.16.4
	k8s.io/node-api => k8s.io/node-api v0.16.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.16.4
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.16.4
	k8s.io/sample-controller => k8s.io/sample-controller v0.16.4
)
