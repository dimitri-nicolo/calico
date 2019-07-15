module github.com/tigera/es-proxy

go 1.11

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v2.4.1-0.20190605125757-c00322e56455+incompatible
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v2.6.0-0.dev.0.20190718111044-9bef69d2b882+incompatible // indirect
)

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.20.0+incompatible // indirect
	github.com/aquasecurity/kube-bench v0.0.29 // indirect
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/go-ini/ini v1.42.0 // indirect
	github.com/go-openapi/spec v0.19.0 // indirect
	github.com/go-playground/locales v0.12.1 // indirect
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/gogo/protobuf v1.2.0 // indirect
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.8-0.20190531063913-f757d8626a73 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kr/pretty v0.1.0 // indirect
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/olivere/elastic v6.2.21+incompatible // indirect
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/projectcalico/felix v3.7.3+incompatible // indirect
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba // indirect
	github.com/projectcalico/go-yaml v0.0.0-20161201183616-955bc3e451ef // indirect
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20161127220527-598e54215bee // indirect
	github.com/projectcalico/libcalico-go v0.0.0
	github.com/prometheus/client_golang v0.9.4 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/smartystreets/goconvey v0.0.0-20190710185942-9d28bd7c0945 // indirect
	github.com/spf13/pflag v0.0.0-20171106142849-4c012f6dcd95 // indirect
	github.com/tigera/calico-k8sapiserver v0.0.0-20190503214445-0e5924229478
	github.com/tigera/compliance v0.0.0-20190708180936-7285e770f61c
	github.com/tigera/licensing v2.2.3+incompatible // indirect
	golang.org/x/crypto v0.0.0-20190611184440-5c40567a22f8 // indirect
	golang.org/x/text v0.3.1-0.20180807135948-17ff2d5776d2 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/appengine v1.4.0 // indirect
	google.golang.org/genproto v0.0.0-20190404172233-64821d5d2107 // indirect
	google.golang.org/grpc v1.21.1 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.29.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.44.0 // indirect
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
	k8s.io/api v0.0.0-20190308202827-072894a440bd
	k8s.io/apimachinery v0.0.0-20190308202827-103fd098999d
	k8s.io/apiserver v0.0.0-20180327025904-5ae41ac86efd // indirect
	k8s.io/client-go v8.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20190709113604-33be087ad058 // indirect
)
