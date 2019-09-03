module github.com/tigera/intrusion-detection/controller

go 1.11

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v2.5.0-mcm0.1.0.20190823023104-d50bfb6c7acc+incompatible

require (
	github.com/araddon/dateparse v0.0.0-20190223010137-262228af701e
	github.com/avast/retry-go v2.2.0+incompatible
	github.com/emicklei/go-restful v2.9.0+incompatible // indirect
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-openapi/spec v0.18.0 // indirect
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/google/go-cmp v0.2.0 // indirect
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/hashicorp/golang-lru v0.5.1
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/json-iterator/go v1.1.6 // indirect
	github.com/kelseyhightower/envconfig v1.3.0 // indirect
	github.com/mailru/easyjson v0.0.0-20190312143242-1de009706dbe // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/olivere/elastic v6.2.16+incompatible
	github.com/onsi/gomega v1.5.0
	github.com/pkg/errors v0.8.1 // indirect
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba // indirect
	github.com/projectcalico/go-yaml v0.0.0-20161201183616-955bc3e451ef // indirect
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20161127220527-598e54215bee // indirect
	github.com/projectcalico/libcalico-go v3.7.0-0.dev.0.20190328155702-d0e07165e343+incompatible
	github.com/prometheus/client_golang v0.9.2 // indirect
	github.com/sirupsen/logrus v1.4.0
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/tigera/calico-k8sapiserver v2.5.0-mcm0.1.0.20190823195257-73c3459a781b+incompatible
	golang.org/x/sync v0.0.0-20181108010431-42b317875d0f
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/api v0.0.0-20180308224125-73d903622b73
	k8s.io/apimachinery v0.0.0-20180228050457-302974c03f7e
	k8s.io/apiserver v0.0.0-20190402105105-9b20910895af // indirect
	k8s.io/client-go v7.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20180216212618-50ae88d24ede // indirect
)
