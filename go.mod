module github.com/tigera/compliance

go 1.16

require (
	github.com/aquasecurity/kube-bench v0.0.34
	github.com/bmizerany/pat v0.0.0-20170815010413-6226ea591a40
	github.com/caimeo/iniflags v0.0.0-20171110233946-ef4ae6c5cd79
	github.com/coreos/go-semver v0.3.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/olivere/elastic/v7 v7.0.22
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/projectcalico/calico v3.21.2+incompatible
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	github.com/tigera/api v0.0.0-20211202170222-d8128d06db71
	github.com/tigera/lma v0.0.0-20220114180745-11c1577ee102
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.5
	k8s.io/apimachinery v0.22.5
	k8s.io/apiserver v0.22.5
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/klog v1.0.0
)

replace (
	github.com/projectcalico/calico => github.com/tigera/calico-private v1.11.0-cni-plugin-private.0.20220114164810-979fc925a331
	github.com/tigera/api => github.com/tigera/calico-private/api v0.0.0-20220114164810-979fc925a331

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
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20211110012726-3cc51fd1e909
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.8
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.8
	k8s.io/kubectl => k8s.io/kubectl v0.21.8
	k8s.io/kubelet => k8s.io/kubelet v0.21.8
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.8
	k8s.io/metrics => k8s.io/metrics v0.21.8
	k8s.io/mount-utils => k8s.io/mount-utils v0.21.8
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.21.8
)
