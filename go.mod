module github.com/tigera/lma

go 1.16

require (
	// This is git hash for tag jose 1.1
	github.com/SermoDigital/jose v0.9.2-0.20161205224733-f6df55f235c2
	github.com/coreos/go-oidc v2.1.0+incompatible
	github.com/go-openapi/runtime v0.19.28
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/olivere/elastic/v7 v7.0.22
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/apiserver v0.0.0-20210421215237-3cfbeeffa901
	github.com/projectcalico/libcalico-go v1.7.2
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.6.1
	github.com/tigera/api v0.0.0-20211202170222-d8128d06db71
	gopkg.in/square/go-jose.v2 v2.5.1
	k8s.io/api v0.21.7
	k8s.io/apimachinery v0.21.7
	k8s.io/apiserver v0.21.7
	k8s.io/client-go v0.21.7
)

replace (
	github.com/projectcalico/apiserver => github.com/tigera/apiserver v0.0.0-20211209181940-47c5f436a856 // indirect
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20211203232041-873deb396265 // indirect

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
