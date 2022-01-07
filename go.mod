module github.com/tigera/intrusion-detection/controller

go 1.16

require (
	github.com/araddon/dateparse v0.0.0-20190223010137-262228af701e
	github.com/avast/retry-go v2.2.0+incompatible
	github.com/olivere/elastic/v7 v7.0.22
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/projectcalico/calico v3.21.2+incompatible
	github.com/simplereach/timeutils v1.2.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tigera/api v0.0.0-20211202170222-d8128d06db71
	github.com/tigera/lma v0.0.0-20220107150026-81a75db8ed35
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	k8s.io/api v0.22.5
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.22.5
	k8s.io/klog/v2 v2.9.0
)

replace (
	github.com/projectcalico/calico => github.com/tigera/calico-private v1.11.0-cni-plugin-private.0.20220107110730-0c545950d39d
	github.com/tigera/api => github.com/tigera/calico-private/api v0.0.0-20220107110730-0c545950d39d

	google.golang.org/grpc => google.golang.org/grpc v1.29.1

	k8s.io/api => k8s.io/api v0.21.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.8
	k8s.io/apiserver => k8s.io/apiserver v0.21.8
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.8
	k8s.io/client-go => github.com/projectcalico/k8s-client-go v0.21.9-0.20220104180519-6bd7ec39553f
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.8
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.8
	k8s.io/code-generator => k8s.io/code-generator v0.21.8
	k8s.io/component-base => k8s.io/component-base v0.21.8
	k8s.io/component-helpers => k8s.io/component-helpers v0.21.8
	k8s.io/controller-manager => k8s.io/controller-manager v0.21.8
	k8s.io/cri-api => k8s.io/cri-api v0.21.8
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.8
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.8
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.8
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.8
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.8
	k8s.io/kubectl => k8s.io/kubectl v0.21.8
	k8s.io/kubelet => k8s.io/kubelet v0.21.8
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.8
	k8s.io/metrics => k8s.io/metrics v0.21.8
	k8s.io/mount-utils => k8s.io/mount-utils v0.21.8
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.8
)
