module github.com/tigera/compliance

go 1.12

replace (
	github.com/emicklei/go-restful => github.com/emicklei/go-restful v0.0.0-20170410110728-ff4f55a20633
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20200130194212-1ef07d41ddd9
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20200130203734-85760318d620
	github.com/tigera/calico-k8sapiserver => github.com/tigera/calico-k8sapiserver v2.7.0-0.dev.0.20200106212250-74a03f23227a+incompatible
	k8s.io/api => k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191114103151-9ca1dc586682
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.0
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.0
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/component-base => k8s.io/component-base v0.17.0
	k8s.io/cri-api => k8s.io/cri-api v0.17.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.0
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.0
	k8s.io/kubectl => k8s.io/kubectl v0.17.0
	k8s.io/kubelet => k8s.io/kubelet v0.17.0
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.0
	k8s.io/metrics => k8s.io/metrics v0.17.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.0
)

require (
	github.com/aquasecurity/kube-bench v0.0.34
	github.com/bmizerany/pat v0.0.0-20170815010413-6226ea591a40
	github.com/caimeo/iniflags v0.0.0-20171110233946-ef4ae6c5cd79
	github.com/coreos/go-semver v0.3.0
	github.com/go-ini/ini v1.48.0 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20190915194858-d3ddacdb130f // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/olivere/elastic/v7 v7.0.6
	github.com/onsi/ginkgo v1.10.2
	github.com/onsi/gomega v1.7.1
	github.com/projectcalico/felix v3.8.5+incompatible
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/tigera/calico-k8sapiserver v2.7.0-0.dev.0.20200106212250-74a03f23227a+incompatible
	github.com/tigera/lma v0.0.0-20200131210249-54bd16ad9d04
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	golang.org/x/time v0.0.0-20190921001708-c4c64cad1fd0 // indirect
	google.golang.org/grpc v1.24.0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/ini.v1 v1.48.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.2.5
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/apiserver v0.17.0
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v1.0.0
)
