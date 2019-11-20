module github.com/tigera/compliance

go 1.12

replace (
	github.com/emicklei/go-restful => github.com/emicklei/go-restful v0.0.0-20170410110728-ff4f55a20633
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20191029225535-3f90fdd6c6ca
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20191113215606-944388dd8364
	github.com/tigera/lma => github.com/tigera/lma v0.0.0-20191120195944-af590c01bfe8
	k8s.io/api => k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20180327025904-5ae41ac86efd
	k8s.io/client-go => k8s.io/client-go v8.0.0+incompatible
)

require (
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/aquasecurity/kube-bench v0.0.34
	github.com/bmizerany/pat v0.0.0-20170815010413-6226ea591a40
	github.com/caimeo/iniflags v0.0.0-20171110233946-ef4ae6c5cd79
	github.com/coreos/go-semver v0.3.0
	github.com/emicklei/go-restful v2.10.0+incompatible // indirect
	github.com/go-ini/ini v1.48.0 // indirect
	github.com/go-openapi/jsonreference v0.19.3 // indirect
	github.com/go-openapi/spec v0.19.3 // indirect
	github.com/gogo/protobuf v1.3.0 // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20190915194858-d3ddacdb130f // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/mailru/easyjson v0.7.0 // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/olivere/elastic/v7 v7.0.6
	github.com/onsi/ginkgo v1.10.2
	github.com/onsi/gomega v1.7.0
	github.com/projectcalico/felix v3.8.2+incompatible
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4 // indirect
	github.com/prometheus/common v0.7.0 // indirect
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/robfig/cron v1.2.0
	github.com/sirupsen/logrus v1.4.2
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/smartystreets/goconvey v0.0.0-20190731233626-505e41936337 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/tigera/calico-k8sapiserver v2.6.0-0.dev.0.20191030050937-be2e3fd6c28a+incompatible
	github.com/tigera/lma v0.0.0-20191030012622-bce3b9ce279b
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	golang.org/x/crypto v0.0.0-20191002192127-34f69633bfdc // indirect
	golang.org/x/net v0.0.0-20191009170851-d66e71096ffb // indirect
	golang.org/x/sys v0.0.0-20191009170203-06d7bd2c5f4f // indirect
	golang.org/x/time v0.0.0-20190921001708-c4c64cad1fd0 // indirect
	google.golang.org/genproto v0.0.0-20191009194640-548a555dbc03 // indirect
	google.golang.org/grpc v1.24.0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/go-playground/validator.v9 v9.30.0 // indirect
	gopkg.in/ini.v1 v1.48.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.0.0-20191009075622-910e671eb668
	k8s.io/apimachinery v0.0.0-20191006235458-f9f2f3f8ab02
	k8s.io/apiserver v0.0.0-20191009120923-2647efb97187
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v0.4.0
	k8s.io/kube-openapi v0.0.0-20190918143330-0270cf2f1c1d // indirect
)
