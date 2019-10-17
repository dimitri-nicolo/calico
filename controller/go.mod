module github.com/tigera/intrusion-detection/controller

go 1.13

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v1.7.2-0.20191015232519-60ba728ac4c5

require (
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/araddon/dateparse v0.0.0-20190223010137-262228af701e
	github.com/avast/retry-go v2.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0 // indirect
	github.com/golang/groupcache v0.0.0-20191002201903-404acd9df4cc // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/hashicorp/golang-lru v0.5.1
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/kelseyhightower/envconfig v1.3.0 // indirect
	github.com/lithammer/dedent v1.1.0
	github.com/olivere/elastic/v7 v7.0.6
	github.com/onsi/gomega v1.7.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/projectcalico/libcalico-go v3.7.0-0.dev.0.20190328155702-d0e07165e343+incompatible
	github.com/simplereach/timeutils v1.2.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/tigera/calico-k8sapiserver v2.6.0-0.dev.0.20191008025816-69c76a2d3f54+incompatible
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0
	golang.org/x/net v0.0.0-20190812203447-cdfb69ac37fc
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	k8s.io/api v0.0.0-20180628040859-072894a440bd
	k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d
	k8s.io/apiserver v0.0.0-20190402105105-9b20910895af // indirect
	k8s.io/client-go v8.0.0+incompatible
)
