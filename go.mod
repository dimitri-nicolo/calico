module github.com/tigera/es-proxy

go 1.11

replace (
	github.com/projectcalico/felix => github.com/tigera/felix-private v2.4.1-0.20190605125757-c00322e56455+incompatible
	github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v0.0.0-20190913172157-338f687c53ee // indirect
	k8s.io/api => k8s.io/api v0.0.0-20180308224125-73d903622b73
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20180228050457-302974c03f7e
	k8s.io/client-go => k8s.io/client-go v7.0.0+incompatible
)

require (
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.20.0+incompatible // indirect
	github.com/aquasecurity/kube-bench v0.0.29 // indirect
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/go-ini/ini v1.42.0 // indirect
	github.com/go-playground/universal-translator v0.16.0 // indirect
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/howeyc/gopass v0.0.0-20190910152052-7cb4b85ec19c // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.8-0.20190531063913-f757d8626a73 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/olivere/elastic/v7 v7.0.6
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/projectcalico/felix v3.7.3+incompatible // indirect
	github.com/projectcalico/libcalico-go master
	github.com/prometheus/client_golang v0.9.4 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/smartystreets/goconvey v0.0.0-20190710185942-9d28bd7c0945 // indirect
	github.com/tigera/calico-k8sapiserver v0.0.0-20190503214445-0e5924229478
	github.com/tigera/compliance v2.6.0-0.dev.0.20190916173451-1ab25cb2ae7d+incompatible
	github.com/tigera/licensing v2.2.3+incompatible // indirect
	github.com/tigera/lma master
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	gopkg.in/go-playground/validator.v9 v9.29.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.44.0 // indirect
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	k8s.io/api v0.0.0-20190831074750-7364b6bdad65
	k8s.io/apimachinery v0.0.0-20190831074630-461753078381
	k8s.io/client-go v8.0.0+incompatible
)
