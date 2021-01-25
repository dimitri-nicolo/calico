module github.com/projectcalico/node

go 1.15

require (
	github.com/kelseyhightower/confd v0.0.0-00010101000000-000000000000
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/pkg/errors v0.9.1
	github.com/projectcalico/cni-plugin v1.11.1-0.20201209190629-11835aaaf9b5
	github.com/projectcalico/felix v0.0.0-20201211230053-74f281689e0c
	github.com/projectcalico/libcalico-go v1.7.2
	github.com/projectcalico/pod2daemon v3.8.2+incompatible // indirect
	github.com/projectcalico/typha v0.7.3-0.20201211220624-6e37e085ee29
	github.com/prometheus/client_golang v1.7.1
	github.com/sirupsen/logrus v1.6.0
	github.com/vishvananda/netlink v1.1.0
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	gopkg.in/fsnotify/fsnotify.v1 v1.4.7
	k8s.io/api v0.19.6
	k8s.io/apimachinery v0.19.6
	k8s.io/client-go v0.19.6
)

replace (
	github.com/Microsoft/hcsshim => github.com/projectcalico/hcsshim v0.8.9-calico
	github.com/bronze1man/goStrongswanVici => github.com/tigera/goStrongswanVici v0.0.0-20180704141420-9b6fdd821dbe
	github.com/kelseyhightower/confd => github.com/tigera/confd-private v1.0.1-0.20210124031052-d0ebb77b4230
	github.com/projectcalico/cni-plugin => github.com/tigera/cni-plugin-private v1.11.1-0.20210124174020-8c03b1a139cc
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20210125221128-87d2bb64a151
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210125220919-50418ce5bfa4
	github.com/projectcalico/typha => github.com/tigera/typha-private v0.6.0-beta1.0.20210123224259-fce3233ac51d

	github.com/sirupsen/logrus => github.com/projectcalico/logrus v1.0.4-calico
	github.com/vishvananda/netlink => github.com/tigera/netlink v0.0.0-20180628131144-3fd955dd6320

	k8s.io/api => k8s.io/api v0.19.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.6
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
