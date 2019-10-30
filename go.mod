module github.com/tigera/es-proxy

go 1.13

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v0.0.0-20191029225535-3f90fdd6c6ca
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20191029214447-cf245bbeba6d // indirect
	k8s.io/api => k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/client-go => k8s.io/client-go v8.0.0+incompatible
)

require (
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.20.0+incompatible // indirect
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/go-ini/ini v1.42.0 // indirect
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/olivere/elastic v6.2.25+incompatible // indirect
	github.com/olivere/elastic/v7 v7.0.6
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/projectcalico/felix v3.7.3+incompatible // indirect
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/sirupsen/logrus v1.4.2
	github.com/smartystreets/goconvey v0.0.0-20190710185942-9d28bd7c0945 // indirect
	github.com/tigera/calico-k8sapiserver v2.5.0-mcm0.1.0.20191030050937-be2e3fd6c28a+incompatible
	github.com/tigera/compliance v2.6.0-0.dev.0.20190916173451-1ab25cb2ae7d+incompatible
	github.com/tigera/lma v0.0.0-20191030012622-bce3b9ce279b
	gopkg.in/go-playground/validator.v9 v9.29.0 // indirect
	gopkg.in/ini.v1 v1.44.0 // indirect
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	k8s.io/api v0.0.0-20190831074750-7364b6bdad65
	k8s.io/apimachinery v0.0.0-20190831074630-461753078381
	k8s.io/apiserver v0.0.0-20190904115329-e72ec4e02467 // indirect
	k8s.io/client-go v8.0.0+incompatible
)
