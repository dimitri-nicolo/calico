module github.com/tigera/lma

go 1.11

replace github.com/projectcalico/libcalico-go => github.com/tigera/libcalico-go-private v2.5.0-mcm0.1.0.20190911005006-fbbc08043d10+incompatible // indirect

require (
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/olivere/elastic/v7 v7.0.6
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba // indirect
	github.com/projectcalico/go-yaml v0.0.0-20161201183616-955bc3e451ef // indirect
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20161127220527-598e54215bee // indirect
	github.com/projectcalico/libcalico-go v1.7.3
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.2
	k8s.io/api v0.0.0-20190831074750-7364b6bdad65
	k8s.io/apimachinery v0.0.0-20190831074630-461753078381
	k8s.io/apiserver v0.0.0-20190904115329-e72ec4e02467
	k8s.io/client-go v0.0.0-20190831074946-3fe2abece89e
)
