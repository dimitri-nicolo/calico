module github.com/tigera/intrusion-detection/controller

go 1.15

require (
	github.com/Microsoft/go-winio v0.4.15 // indirect
	github.com/araddon/dateparse v0.0.0-20190223010137-262228af701e
	github.com/avast/retry-go v2.2.0+incompatible
	github.com/containerd/containerd v1.4.4 // indirect
	github.com/containernetworking/cni v0.8.0 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v20.10.2+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/olivere/elastic/v7 v7.0.9-0.20191104165744-604114ea2c85
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/projectcalico/libcalico-go v1.7.2
	github.com/simplereach/timeutils v1.2.0 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/tigera/api v0.0.0-20210826165108-d6e4592c976a
	github.com/tigera/licensing v1.0.1-0.20210629175442-f7d453f3e818
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0
	golang.org/x/net v0.0.0-20210224082022-3d97a244fca7
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
	k8s.io/klog/v2 v2.8.0
)

replace (
	github.com/projectcalico/apiserver => github.com/tigera/apiserver v0.0.0-20210831002403-355d03a5ebc2
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20210831001605-43dc71ec85fd
	github.com/tigera/licensing => github.com/tigera/licensing v1.0.1-0.20210805040145-adb68d9c078b

	// k8s apiserver upgrade
	k8s.io/api => k8s.io/api v0.21.0-rc.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.0-rc.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.0-rc.0
	k8s.io/apiserver => k8s.io/apiserver v0.21.0-rc.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.0-rc.0
	k8s.io/client-go => k8s.io/client-go v0.21.0-rc.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.0-rc.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.0-rc.0
	k8s.io/code-generator => k8s.io/code-generator v0.21.0-rc.0
	k8s.io/component-base => k8s.io/component-base v0.21.0-rc.0
	k8s.io/component-helpers => k8s.io/component-helpers v0.21.0-rc.0
	k8s.io/controller-manager => k8s.io/controller-manager v0.21.0-rc.0
	k8s.io/cri-api => k8s.io/cri-api v0.21.0-rc.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.0-rc.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.0-rc.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.0-rc.0
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.0-rc.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.0-rc.0
	k8s.io/kubectl => k8s.io/kubectl v0.21.0-rc.0
	k8s.io/kubelet => k8s.io/kubelet v0.21.0-rc.0
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.0-rc.0
	k8s.io/metrics => k8s.io/metrics v0.21.0-rc.0
	k8s.io/mount-utils => k8s.io/mount-utils v0.21.0-rc.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.0-rc.0
)
