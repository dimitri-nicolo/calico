module github.com/tigera/lma

go 1.15

replace (
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210106003228-1dadd938e2cb // indirect
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
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.18.12
)

require (
	github.com/coreos/go-oidc v2.1.0+incompatible
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/olivere/elastic/v7 v7.0.6
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/projectcalico/libcalico-go v1.7.2
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	github.com/tigera/apiserver v0.0.0-20201221172353-3c54b7106abc
	gopkg.in/square/go-jose.v2 v2.2.3-0.20190111193340-cbf0fd6a984a
	k8s.io/api v0.18.12
	k8s.io/apimachinery v0.18.12
	k8s.io/apiserver v0.18.12
	k8s.io/client-go v0.18.12
	k8s.io/kubernetes v1.18.12
)
